package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"strings"
	"time"
)

func parsePayPeriodArgs(args []string) (*BillSchema, error) {
	if len(args) < 3 {
		return nil, errors.New(fmt.Sprintf(
			"expect at least 3, CLIENT FROM TO [-n NOTE], but got %d args", len(args)))
	}

	client := strings.TrimSpace(args[0])
	if !isValidClient(client) {
		return nil, errors.New(fmt.Sprintf("invalid CLIENT: '%s'", client))
	}

	fromStamp, e := strconv.ParseInt(strings.TrimSpace(args[1]), 10, 64)
	if e != nil {
		return nil, errors.New(fmt.Sprintf("expected FROM to be int: %s", e))
	}

	toStamp, e := strconv.ParseInt(strings.TrimSpace(args[2]), 10, 64)
	if e != nil {
		return nil, errors.New(fmt.Sprintf("expected FROM to be int: %s", e))
	}

	if fromStamp >= toStamp {
		return nil, errors.New("expected FROM to be younger stamp than TO")
	}

	var note string
	if len(args) > 3 {
		note = strings.TrimSpace(strings.Join(args[3:], " "))
	}

	return &BillSchema{
		Endclusive:   time.Unix(toStamp, 0 /*nanoseconds*/),
		Startclusive: time.Unix(fromStamp, 0 /*nanoseconds*/),
		Project:      client,
		Note:         note,
	}, nil
}

func commitPayPeriod(db *sql.DB, b *BillSchemaSQL) error {
	stmt, e := db.Prepare(`
		INSERT INTO
		paychecks(endclusive, startclusive, project, note)
		VALUES (?, ?, ?, ?)
	`)
	if e != nil {
		return e
	}

	// TODO(zacsh) expose result val here via debug flags on cli
	_, e = stmt.Exec(b.Endclusive, b.Startclusive, b.Project, b.Note)

	return e
}

func markPayPeriod(dbPath string, args []string) error {
	bill, e := parsePayPeriodArgs(args)
	if e != nil {
		return errors.New(fmt.Sprintf("parse args: %s", e))
	}

	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return errors.New(fmt.Sprintf("bil sql: %s", e))
	}
	defer db.Close()

	return commitPayPeriod(db, bill.toSQL())
}
