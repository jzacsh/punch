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

func seekStillOpenPunchIn(db *sql.DB, cmd *SeekCmd) error {
	if cmd.SeekTo.Before(cmd.StillOpen) {
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
			// TODO(zacsh) add a CLI flag to break this ambiguity
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
	closingPunch.Note = "" // TODO(zacsh) add CLI flag to accept note
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
	return nil
}

func seekExistingPunchOut(db *sql.DB, cmd *SeekCmd) error {
	if cmd.SeekTo.Sub(cmd.Faulty) == 0 {
		return fmt.Errorf("no effective change requested: FAULTY_STAMP equals SEEK_TO")
	}

	rows, e := db.Query(`
		SELECT * FROM punchcard
		WHERE punch IS ?
		AND status IS 0
	`, cmd.Faulty.Unix())
	if e != nil {
		return fmt.Errorf("querying for FAULTY_STAMP: %s", e)
	}
	defer rows.Close()

	var origClose *CardSchema
	for rows.Next() {
		card, e := scanToCard(rows)
		if e != nil {
			return fmt.Errorf("reading FAULTY_STAMP punch: %s", e)
		}
		if origClose != nil {
			// TODO(zacsh) add a CLI flag to break this ambiguity
			return fmt.Errorf("ambiguous: more than one punch-out shares this FAULTY_STAMP")
		}
		origClose = card
	}
	if origClose == nil {
		return fmt.Errorf("No punches found matching FAULTY_STAMP")
	}

	newClose := *origClose
	newClose.Punch = cmd.SeekTo

	seekDirection := "Rewind"
	seekOffset := origClose.Punch.Sub(cmd.SeekTo)
	if seekOffset < 0 {
		seekDirection = "Fast-forward"
		seekOffset = cmd.SeekTo.Sub(origClose.Punch)
	}
	fmt.Printf("%sing '%s' session's close by %s\n",
		seekDirection, newClose.Project, seekOffset)

	rows, e = db.Query(`
		SELECT * FROM punchcard
		WHERE punch < ?
		AND project IS ?
		AND status IS 1
		LIMIT 1
	`, cmd.Faulty.Unix(), origClose.Project)
	if e != nil {
		return fmt.Errorf("querying for FAULTY_STAMP's opening punch: %s", e)
	}

	var punchIn *CardSchema
	for rows.Next() {
		punchIn, e = scanToCard(rows)
		if e != nil {
			return fmt.Errorf("reading FAULTY_STAMP's session cards: %s", e)
		}
	}
	if punchIn == nil {
		return fmt.Errorf("bad data state: no open punch to FAULTY_STAMP's close")
	}

	if !punchIn.Punch.Before(cmd.SeekTo) {
		return fmt.Errorf(
			"SEEK_TO will rewind sesion-close to %s BEFORE session's start",
			punchIn.Punch.Sub(cmd.SeekTo))
	}

	if cmd.IsDryRun {
		fmt.Fprint(os.Stderr, "[-d]ry-run: finishing early; NO changes written\n")
		return nil
	}

	stmt, e := db.Prepare(`
		UPDATE punchcard
		SET punch = ?
		WHERE punch IS ?
		AND project IS ?
	`)
	if e != nil {
		return fmt.Errorf("building UPDATE query: %s", e)
	}

	// TODO(zacsh) expose result val here via debug flags on cli
	if _, e := stmt.Exec(
		newClose.Punch.Unix(),
		origClose.Punch.Unix(),
		newClose.Project); e != nil {
		return fmt.Errorf("running UPDATE query: %s", e)
	}
	return nil
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
		if e := seekStillOpenPunchIn(db, cmd); e != nil {
			return e
		}
	} else {
		if e := seekExistingPunchOut(db, cmd); e != nil {
			return e
		}
	}

	fmt.Println("Done.")
	return nil
}
