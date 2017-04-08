package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
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

func queryClient(db *sql.DB, client string) error {
	rows, e := db.Query(`
		SELECT * FROM punchcard
		WHERE project IS ?
		ORDER BY punch ASC;
	`, client)
	if e != nil {
		return e
	}
	defer rows.Close()

	var numSessions int
	var total time.Duration
	fmt.Printf("Report on '%s':\n", client)
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

			outPunchFormat := "15:04:05.9999 -0700 MST"
			if session.Duration > time.Hour*22 {
				outPunchFormat = "01-02" + outPunchFormat
			}
			fmt.Printf("  %s from %s to %s",
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
		fmt.Printf("Warning: no records found for this client string.\n")
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

// TODO(zacsh) print notice of timezone in parenthesis in header, then exclude
// from record output
func queryDump(db *sql.DB) error {
	rows, e := db.Query(`SELECT * FROM punchcard ORDER BY punch ASC;`)
	if e != nil {
		return e
	}
	defer rows.Close()

	fmt.Printf("Punch, Status, Project, Note\n")
	for rows.Next() {
		punch, e := scanToCard(rows)
		if e != nil {
			return e
		}
		fmt.Printf(
			"%s, %s, %s, %s\n",
			punch.Punch,
			fromStatus(punch.IsStart),
			punch.Project,
			fromNote(punch.Note))
	}

	return nil
}

func queryStatus(db *sql.DB) error {
	// TODO do JOIN or something to get ONE row PER group (per project); currently
	// this will erronesouly show only one punched-in status, even if punched into
	// multiple projects
	rows, e := db.Query(`
		SELECT * FROM punchcard
		ORDER BY punch ASC
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
		fmt.Printf(
			"%s: %s so far\n",
			punch.Project,
			durationToStr(time.Since(punch.Punch)))
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
		queryClient(db, args[1])
	default:
		return errors.New(fmt.Sprintf(
			"usage error: unrecognized query cmd, '%s'", args[0]))
	}

	return nil
}
