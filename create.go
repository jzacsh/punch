package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"

	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

func ensureUserWantsAutocreation(dbPath string) error {
	fmt.Printf(
		"$PUNCH_CARD database not yet created\n\t%s\n", dbPath)
	fmt.Printf("Should one be automatically started now? [y/N] ")

	reader := bufio.NewReader(os.Stdin)
	nextLine, e := reader.ReadString('\n')
	if e != nil {
		return fmt.Errorf("response parsing: %s", e)
	}
	response := strings.TrimSpace(nextLine)
	if len(response) < 1 || strings.ToLower(string(response[0])) != "y" {
		return errors.New("auto-creation offer rejected")
	}
	return nil
}

func subCmdCreate(dbPath string) error {
	if e := ensureUserWantsAutocreation(dbPath); e != nil {
		return e
	}

	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return fmt.Errorf("error opening sqlite3: %s", e)
	}

	stmt, e := db.Prepare(`
CREATE TABLE punchcard (
  punch       INTEGER NOT NULL PRIMARY KEY,
  status      INTEGER NOT NULL,
  project     TEXT NOT NULL,
  note        TEXT
);
	`)
	if e != nil {
		return fmt.Errorf("preparing punchcard table: %s", e)
	}
	if _, e := stmt.Exec(); e != nil {
		return fmt.Errorf("creating punchcard table: %s", e)
	}

	stmt, e = db.Prepare(`
CREATE TABLE paychecks (
  endclusive   INTEGER NOT NULL PRIMARY KEY,
  startclusive INTEGER NOT NULL,
  project      TEXT NOT NULL,
  note         TEXT
);
	`)
	if e != nil {
		return fmt.Errorf("preparing paychecks table: %s", e)
	}
	if _, e := stmt.Exec(); e != nil {
		return fmt.Errorf("creating paychecks table: %s", e)
	}

	fmt.Print(`Empty tables successfully created.

  To start keep records try 'punch' and 'query' commands.
	For reminders of their arguments, see '-h'.
	For a listing of ALL commands and full docs, see 'help'
	For a reminder of what one command does, see 'help [cmd]', eg: 'help punch'
`)
	return nil
}
