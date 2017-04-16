package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"strings"
	"time"
)

func subCmdAmend(dbPath string, args []string) error {
	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return fmt.Errorf("punch cards: %s", e)
	}
	defer db.Close()

	if len(args) < 1 {
		return fmt.Errorf("argument TARGET_STAMP is required")
	}

	targetStamp, e := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
	if e != nil {
		return fmt.Errorf("parsing unix timestamp, TARGET_STAMP ('%s'), %s", args[0], e)
	}

	target := time.Unix(targetStamp, 0 /*nanoseconds*/)

	var replacement string
	if len(args) > 1 {
		replacement = strings.TrimSpace(strings.Join(args[1:], " "))
	}
	isDeletion := len(replacement) < 1
	return fmt.Errorf(
		"NOTE amendment not yet implemented, but got:\n\t Note: '%s'\n\tis deletion: %t\n\twhich: %s (@%s)\n",
		replacement,
		isDeletion,
		target,
		targetStamp)
}
