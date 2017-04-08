package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"math"
	"os"
	"strconv"
	"time"
)

const queryDefaultCmd string = "status"

func scanToCard(rows *sql.Rows) (*CardSchema, error) {
	raw := &CardSchemaRaw{}
	if e := rows.Scan(&raw.Punch, &raw.Status, &raw.Project, &raw.Note); e != nil {
		return nil, e
	}
	return raw.toCard(), nil
}

func isEmptyTime(t *time.Time) bool {
	var defaultTime time.Time
	return t.Sub(defaultTime) == 0
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
	fmt.Printf("Report on '%s' (in %s)%s:\n", client, getTZContext(), limited)
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

			outPunchFormat := "15:04:05.9999"
			if session.Duration > time.Hour*22 {
				outPunchFormat = "01-02" + outPunchFormat
			}
			fmt.Printf(fmt.Sprintf("%s%d%s", "%", durationToStrMaxLen, "s from %s to %s"),
				session.durationToStr(),
				session.StartAt.Format("2006-01-02 15:04:05.99999"),
				session.StopAt.Format(outPunchFormat))
			if len(session.NoteStart) > 0 {
				fmt.Printf(" %s", session.NoteStart)
			}
			if len(session.NoteStop) > 0 {
				fmt.Printf(" %s", session.NoteStop)
			}
			fmt.Printf("\n")
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

	if len(punches) > 2 {
		fmt.Printf("Summary: Worked %s over %d sessions\n", total, numSessions)
	} else {
		var fromClause string
		if !isEmptyTime(from) {
			fromClause = fmt.Sprintf(" in the past %s", time.Since(*from))
		}
		fmt.Printf("Warning: no records found for this client%s.\n", fromClause)
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
	// TODO do JOIN or something to get ONE row PER group (per project); currently
	// this will erronesouly show only one punched-in status, even if punched into
	// multiple projects
	rows, e := db.Query(`
		SELECT * FROM punchcard
		ORDER BY punch DESC
		LIMIT 1;
	`)
	if e != nil {
		return e
	}
	defer rows.Close()

	for rows.Next() {
		punch, e := scanToCard(rows)
		if e != nil {
			return e
		}
		if punch.IsStart {
			fmt.Printf(
				"%s: %s so far\n",
				punch.Project,
				durationToStr(time.Since(punch.Punch)))
		} else {
			fmt.Fprintf(os.Stderr, "Not on the clock.\n")
		}
	}
	return nil
}

func cardQuery(dbInfo os.FileInfo, dbPath string, args []string) error {
	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return errors.New(fmt.Sprintf("punch cards: %s", e))
	}
	defer db.Close()

	if len(args) == 0 {
		return queryDump(db)
	}

	switch args[0] {
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
				return errors.New(fmt.Sprintf("parsing FROM_STAMP: %s", e))
			}
			from = time.Unix(fromStamp, 0 /*nanoseconds*/)
		}
		queryClient(db, args[1], &from)
	default:
		return errors.New(fmt.Sprintf(
			"usage error: unrecognized query cmd, '%s'", args[0]))
	}

	return nil
}
