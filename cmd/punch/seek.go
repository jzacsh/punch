package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"time"
)

type SeekCmd struct {
	SeekTo    time.Time
	Faulty    time.Time
	StillOpen time.Time
	IsDryRun  bool
}

func (s *SeekCmd) isClose() bool { return !s.StillOpen.IsZero() }

func parseSeekCmd(args []string) (*SeekCmd, error) {
	cmd := &SeekCmd{}
	if len(args) < 2 {
		return nil, fmt.Errorf(
			"expected at least SEEK_TO and one more stamp, got %d args",
			len(args))
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-d":
			cmd.IsDryRun = true
		case "-c":
			i++ // skip to next arg
			stamp, e := parseStampCommand(args[i])
			if e != nil {
				return nil, fmt.Errorf("STILL_OPEN: %s", e)
			}
			cmd.StillOpen = stamp
		default:
			// we're processing a positional argument, a timestamp
			if cmd.SeekTo.IsZero() {
				stamp, e := parseStampCommand(args[i])
				if e != nil {
					return nil, fmt.Errorf("SEEK_TO: %s", e)
				}
				cmd.SeekTo = stamp
			} else {
				stamp, e := parseStampCommand(args[i])
				if e != nil {
					return nil, fmt.Errorf("FAULTY_STAMP: %s", e)
				}
				cmd.Faulty = stamp
			}
		}
	}

	if cmd.SeekTo.IsZero() {
		return nil, fmt.Errorf("require positional arg SEEK_TO")
	}

	return cmd, nil
}

func subCmdSeek(dbPath string, args []string) error {
	cmd, e := parseSeekCmd(args)
	if e != nil {
		return fmt.Errorf("parsing command: %s", e)
	}

	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return fmt.Errorf("delete from db: %s", e)
	}
	defer db.Close()

	if cmd.isClose() {
		if cmd.SeekTo.Unix() <= cmd.StillOpen.Unix() {
			return fmt.Errorf("SEEK_TO <= STILL_OPEN creates empty session")
		}
		rows, e := db.Query(`
			SELECT * FROM punchcard
			WHERE status IS 1
			AND punch IS ?
		`, cmd.StillOpen.Unix())
		if e != nil {
			return fmt.Errorf("querying STILL_OPEN punch: %s", e)
		}
		defer rows.Close()

		var openPunch *CardSchema
		for rows.Next() {
			card, e := scanToCard(rows)
			if e != nil {
				return fmt.Errorf("reading STILL_OPEN punch: %s", e)
			}
			if openPunch != nil {
				// TODO(zacsh) add a flag to break this ambiguity
				return fmt.Errorf(
					"ambiguous: more than one client has open seesion starting at STILL_OPEN")
			}
			openPunch = card
		}
		if openPunch == nil {
			return fmt.Errorf("No punches found matching STILL_OPEN")
		}

		closingPunch := *openPunch
		closingPunch.IsStart = false
		closingPunch.Punch = cmd.SeekTo
		resultingSession := openPunch.toSession(&closingPunch)
		fmt.Printf(
			"Closing '%s' session, resulting in:\n%s\n",
			closingPunch.Project, resultingSession)
		if cmd.IsDryRun {
			fmt.Fprint(os.Stderr, "[-d]ry-run: finishing early; NO changes written\n")
			return nil
		}
		if e := punchProject(db, closingPunch.toSQL()); e != nil {
			return fmt.Errorf("closing session: %s", e)
		}
	} else {
		// TODO(zacsh): assert: cmd.SeekTo > [SQL: faulty's opening-punch]
		return fmt.Errorf("bug: not yet implemented")
	}

	fmt.Println("Done.")
	return nil
}
