package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

func scanToCard(rows *sql.Rows) (*CardSchema, error) {
	raw := &CardSchemaSQL{}
	if e := rows.Scan(&raw.Punch, &raw.Status, &raw.Project, &raw.Note); e != nil {
		return nil, e
	}
	return raw.toCard(), nil
}

func scanToBill(rows *sql.Rows) (*BillSchema, error) {
	raw := &BillSchemaSQL{}
	e := rows.Scan(&raw.Endclusive, &raw.Startclusive, &raw.Project, &raw.Note)
	if e != nil {
		return nil, e
	}
	return raw.toBill(), nil
}

func queryClient(db *sql.DB, client string, from *time.Time) error {
	var fromStamp int64
	if !isEmptyTime(from) {
		fromStamp = from.Unix()
	}

	rows, e := db.Query(`
		SELECT * FROM punchcard
		WHERE project IS ?
		AND punch > ?
		ORDER BY punch ASC;
	`, client, fromStamp)
	if e != nil {
		return e
	}
	defer rows.Close()

	var limited string
	if !isEmptyTime(from) {
		limited = fmt.Sprintf(" from %s", from.Format(format_dateTime))
	}

	var numSessions int
	var total time.Duration
	fmt.Printf("Sessions on '%s' (in %s)%s:\n", client, getTZContext(), limited)
	var punches []*CardSchema
	numRecords := 0
	for rows.Next() {
		card, e := scanToCard(rows)
		if e != nil {
			return e
		}

		numRecords++
		if numRecords == 1 && !card.IsStart {
			fmt.Printf(
				"  [ERROR: stray punch-out!] at %s (note: '%s')\n",
				card.Punch.Unix(), fromNote(card.Note))
			continue
		} else if !card.IsStart {
			session := punches[len(punches)-1].toSession(card)
			numSessions++
			total += session.Duration

			fmt.Printf("%s\n", session)
		}

		punches = append(punches, card)
	}

	if len(punches)%2 == 1 {
		accumulating := time.Since(punches[len(punches)-1].Punch)
		total += accumulating
		fmt.Printf(
			"Note: currently punched-in & working; %s so far\n",
			accumulating)
	}

	if len(punches) >= 2 {
		fmt.Printf("Summary: Worked %s over %d sessions\n", total, numSessions)
	} else {
		var fromClause string
		if !isEmptyTime(from) {
			fromClause = fmt.Sprintf(" in the past %s", time.Since(*from))
		}
		whatNotFound := "sessions"
		if len(punches) == 0 && isEmptyTime(from) {
			whatNotFound = "records" // we found _NOTHING_ and no FROM clause passed
		}
		fmt.Printf("Warning: no %s found for this client%s.\n", whatNotFound, fromClause)
	}

	return nil
}

func queryClients(db *sql.DB) error {
	rows, e := db.Query(`
		SELECT DISTINCT(project) as project
		FROM punchcard ORDER BY project ASC;
	`)
	if e != nil {
		return e
	}
	defer rows.Close()

	for rows.Next() {
		var client string
		if e := rows.Scan(&client); e != nil {
			return e
		}
		fmt.Printf("%s\n", client)
	}

	return nil
}

func countProjects(db *sql.DB) (int, error) {
	rows, e := db.Query(`SELECT COUNT(DISTINCT project) FROM punchcard;`)
	if e != nil {
		return 0, e
	}
	defer rows.Close()
	for rows.Next() {
		var count int
		if e := rows.Scan(&count); e != nil {
			return 0, e
		}
		return count, nil
	}

	return 0, nil // zero projects found
	// TODO double check this would result in empty rows.Next()
}

func queryDump(db *sql.DB) error {
	rows, e := db.Query(`SELECT * FROM punchcard ORDER BY punch ASC;`)
	if e != nil {
		return e
	}
	defer rows.Close()

	var longestProjectStr float64

	lastPunchInFor := make(map[string]CardSchema)
	sessionsFor := make(map[string][]Session)
	fmt.Printf("Punch [%s], Status, Project, Note\n", getTZContext())
	for rows.Next() {
		punch, e := scanToCard(rows)
		if e != nil {
			return e
		}
		fmt.Printf(
			"%s, %3s, %s, %s\n",
			punch.Punch.Format(format_dateTime),
			fromStatus(punch.IsStart),
			punch.Project,
			fromNote(punch.Note))

		longestProjectStr = math.Max(float64(len(punch.Project)), longestProjectStr)

		if punch.IsStart {
			lastPunchInFor[punch.Project] = *punch
		} else {
			lastPunch := lastPunchInFor[punch.Project]
			sessionsFor[punch.Project] = append(
				sessionsFor[punch.Project],
				*((&lastPunch).toSession(punch)))
			lastPunchInFor[punch.Project] = emptyCard
		}
	}

	if longestProjectStr == 0 {
		return fmt.Errorf("zero punch-card records found")
	}

	// Summarize above dump
	fmt.Printf("\nProject, Sessions, Status, Worked Time\n")
	for project, sessions := range sessionsFor {
		var total time.Duration
		for _, session := range sessions {
			total += session.Duration
		}

		status := "n/a"
		last := lastPunchInFor[project]
		if !last.isEmptyCard() && last.IsStart {
			status = "WORKING"
		}
		fmt.Printf(
			fmt.Sprintf("%s+%d%s\n", "%", int(longestProjectStr), "s, %4d, %s, %s"),
			project, len(sessions), status, durationToStr(total))
	}

	return nil
}

func queryStatus(db *sql.DB) error {
	rows, e := db.Query(`
		SELECT * FROM punchcard
		GROUP BY project
		ORDER BY punch DESC;
	`)
	if e != nil {
		return e
	}
	defer rows.Close()

	isOnClock := false
	for rows.Next() {
		punch, e := scanToCard(rows)
		if e != nil {
			return e
		}
		if punch.IsStart {
			isOnClock = true
			fmt.Printf(
				"%s: %s so far\n",
				punch.Project,
				durationToStr(time.Since(punch.Punch)))
			// TODO include *total* since-last-payperiod logged, in parenthesis, eg:
			// "golangpunch: 0:03 so far (37:14:00 since last bill)"
		}
	}

	if isOnClock {
		return nil
	}
	return fmt.Errorf("not on the clock")
}

// TODO figure out how to pass entire `clients` array as-is into var-args
// db.Exec(), instead of manually building a query string with our own
// injection-checking
func hack_queryPaychecksIn(db *sql.DB, clients []string) (*sql.Rows, error) {
	query := "SELECT * FROM paychecks\n"
	if len(clients) > 0 {
		for i, c := range clients {
			if !isValidClient(c) {
				return nil, fmt.Errorf("invalid client: '%s'", c)
			}
			clients[i] = strings.TrimSpace(c)
		}
		query += fmt.Sprintf(
			"WHERE project IN ('%s')\n", strings.Join(clients, "', '"))
	}
	query += "ORDER BY endclusive ASC;"

	return db.Query(query)
}

func queryBills(db *sql.DB, clients []string) error {
	// TODO(zacsh) make this a JOIN and fetch all the punches within a
	// {end,start}clusive, and include amount of time worked in this report
	//   SELECT *
	//   FROM paychecks as p
	//   JOIN punchcard as c
	//   ON p.project=c.project
	//   AND p.startclusive < c.punch
	//   AND p.endclusive > c.punch;

	rows, e := hack_queryPaychecksIn(db, clients)
	if e != nil {
		return e
	}
	defer rows.Close()

	foundPayPeriod := false
	fmt.Printf("Billed, From (%s), To, Note\n", getTZContext())
	for rows.Next() {
		b, e := scanToBill(rows)
		if e != nil {
			return e
		}
		foundPayPeriod = true
		fmt.Println(b.String(false /*showTimezone*/))
	}
	if !foundPayPeriod {
		fmt.Printf("No pay-periods closed, yet.\n")
	}
	return nil
}

// Subcommand "query" driver; has it own subcommands `args` which drive its
// response
func subCmdQuery(dbInfo os.FileInfo, dbPath string, args []string) error {
	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return fmt.Errorf("punch cards: %s", e)
	}
	defer db.Close()

	subCmd := "dump"
	if len(args) > 0 {
		subCmd = strings.TrimSpace(args[0])
	}

	switch subCmd {
	case "bill", "bills":
		var clients []string
		if len(args) > 1 {
			clients = args[1:]
		}
		return queryBills(db, clients)
	case "status":
		return queryStatus(db)
	case "list":
		return queryClients(db)
	case "report":
		if len(args) < 2 || len(args[1]) < 1 {
			return errors.New("usage error: need client name to report on")
		}
		var from time.Time
		if len(args) > 2 {
			fromStamp, e := strconv.ParseInt(args[2], 10, 64)
			if e != nil {
				return fmt.Errorf("parsing FROM_STAMP: %s", e)
			}
			from = time.Unix(fromStamp, 0 /*nanoseconds*/)
		}
		queryClient(db, args[1], &from)
	case "dump":
		return queryDump(db)
	default:
		return fmt.Errorf(
			"usage error: unrecognized query cmd, '%s'", subCmd)
	}

	return nil
}
