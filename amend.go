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

	// TODO make this interactive (with a -q(uiet) flag to not ask)

	return fmt.Errorf(
		"NOTE amendment not yet implemented, but got:\n\t Note: '%s'\n\tis deletion: %t\n\twhich: %s (@%s)\n",
		note,
		isDeletion,
		target,
		target.Unix())
}
