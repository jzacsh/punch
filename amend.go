package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"strings"
	"time"
)

// TARGET_STAMP, [NOTE], error
func parseAmendCli(args []string) (time.Time, string, error) {
	var target time.Time
	if len(args) < 1 {
		return target, "", fmt.Errorf("argument TARGET_STAMP is required")
	}

	targetStamp, e := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
	if e != nil {
		return target, "", fmt.Errorf("parsing unix timestamp, TARGET_STAMP ('%s'), %s", args[0], e)
	}

	target = time.Unix(targetStamp, 0 /*nanoseconds*/)

	var replacement string
	if len(args) > 1 {
		replacement = strings.TrimSpace(strings.Join(args[1:], " "))
	}

	return target, replacement, nil
}

func subCmdAmend(dbPath string, args []string) error {
	target, note, e := parseAmendCli(args)
	if e != nil {
		return e
	}
	isDeletion := len(note) < 1

	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return fmt.Errorf("punch cards: %s", e)
	}
	defer db.Close()

	noteAction := "update"
	if isDeletion {
		noteAction = "delete"
	}

	stmt, e := db.Prepare(`
		UPDATE punchcard
		SET note = ?
		WHERE punch = ?
	;`)
	if e != nil {
		return fmt.Errorf("preparing db modification: %s", e)
	}

	// TODO make this interactive (with a -q(uiet) flag to not ask)
	r, e := stmt.Exec(note, target.Unix())
	if e != nil {
		return fmt.Errorf("trying to %s note: %s", noteAction, e)
	}
	a, e := r.RowsAffected()
	if e != nil {
		return fmt.Errorf("trying to parse results of %s: %s", noteAction, e)
	}

	if a != 1 {
		return fmt.Errorf("expected 1 punch record affected, but got %d", a)
	}

	fmt.Printf(
		"Done: successfully %sd note on %s punch\n",
		noteAction, target.Format(format_dateTime))
	return nil
}
