package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strconv"
	"strings"
	"time"
)

func getImpliedToStamp(db *sql.DB, client string) (int64, error) {
	rows, e := db.Query(`
		SELECT * FROM punchcard
		WHERE project IS ?
		ORDER BY punch DESC
		LIMIT 2;
	`, client)
	if e != nil {
		return 0, e
	}
	defer rows.Close()

	for rows.Next() {
		c, e := scanToCard(rows)
		if e != nil {
			return 0, e
		}
		if c.IsStart {
			continue
		}

		return c.Punch.Unix(), nil
	}

	return 0, fmt.Errorf(
		"implied TO stamp, but no full work records found", client)
}

func getImpliedFromStamp(db *sql.DB, client string) (int64, error) {
	rows, e := db.Query(`
		SELECT * FROM paychecks
		WHERE project IS ?
		ORDER BY endclusive DESC
		LIMIT 1;
	`, client)
	if e != nil {
		return 0, e
	}
	defer rows.Close()

	for rows.Next() {
		b, e := scanToBill(rows)
		if e != nil {
			return 0, e
		}

		return b.Endclusive.Unix(), nil
	}

	// If here, then no previous paycheck, so *all* of history is implied
	// beginning of paycheck...

	rows, e = db.Query(`
		SELECT * FROM punchcard
		WHERE project IS ?
		ORDER BY punch ASC
		LIMIT 1;
	`, client)
	if e != nil {
		return 0, e
	}
	defer rows.Close() // TODO funky to have two rows.Close() deferals?

	for rows.Next() {
		c, e := scanToCard(rows)
		if e != nil {
			return 0, e
		}

		return c.Punch.Unix(), nil
	}

	return 0, fmt.Errorf(
		"implied '%s' FROM impossible without work or payperiod history", client)
}

func parsePayPeriodArgs(db *sql.DB, args []string) (bool, *BillSchema, error) {
	isDryRun := false

	client := strings.TrimSpace(args[0])
	if !isValidClient(client) {
		return isDryRun, nil, fmt.Errorf("invalid CLIENT: '%s'", client)
	}

	isImpliedFrom := true
	isImpliedTo := true

	var e error
	var note string
	var fromStamp, toStamp int64
	if len(args) > 1 {
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "-n":
				noteStartIdx := i + 1
				note = strings.TrimSpace(strings.Join(args[noteStartIdx:], " "))
				if len(note) < 1 {
					return isDryRun, nil, errors.New("-n passed, but no NOTE found.")
				}
				i = len(args) // end for loop

			case "-d":
				isDryRun = true

			case "-f":
				fromStamp, e = strconv.ParseInt(strings.TrimSpace(args[i+1]), 10, 64)
				if e != nil {
					return isDryRun, nil, fmt.Errorf(
						"bad FROM timestamp, '%s'", args[i+1])
				}
				isImpliedFrom = false
				i++ // skip FROM stamp

			case "-t":
				toStamp, e = strconv.ParseInt(strings.TrimSpace(args[i+1]), 10, 64)
				if e != nil {
					return isDryRun, nil, fmt.Errorf(
						"bad TO timestamp, '%s'", args[i+1])
				}
				isImpliedTo = false
				i++ // skip TO stamp

			default:
				return isDryRun, nil, fmt.Errorf(
					"unrecognized commandline at '%s'", args[i:])
			}
		}
	}

	if isImpliedFrom {
		fromStamp, e = getImpliedFromStamp(db, client)
		if e != nil {
			return isDryRun, nil, e
		}
	}

	if isImpliedTo {
		toStamp, e = getImpliedToStamp(db, client)
		if e != nil {
			return isDryRun, nil, e
		}
	}

	if fromStamp >= toStamp {
		return isDryRun, nil, errors.New("expected FROM to be older stamp than TO")
	}

	return isDryRun, &BillSchema{
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

func subCmdBill(dbPath string, args []string) error {
	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return fmt.Errorf("bill sql: %s", e)
	}
	defer db.Close()

	isDryRun, bill, e := parsePayPeriodArgs(db, args)
	if e != nil {
		return fmt.Errorf("parse args: %s", e)
	}

	var note string
	if len(bill.Note) > 0 {
		note = fmt.Sprintf(`With NOTE:
    %s`, bill.Note)
	}

	fmt.Fprintf(os.Stderr, `    Will create bill for '%s':
      from '%s'
      to   '%s'
    %s%s`,
		bill.Project,
		bill.Startclusive,
		bill.Endclusive,
		note,
		"\n")

	if isDryRun {
		fmt.Fprintf(os.Stderr, "\n[-d]ry-run mode; NOT writing any changes\n")
		return nil
	}

	e = commitPayPeriod(db, bill.toSQL())
	if e == nil {
		fmt.Fprintf(os.Stderr, "Done.\n")
	}
	return e
}
