package main

import (
	sql "database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

type payPeriod struct {
	End     int
	Start   int
	Project string
	Note    string
}

func (p payPeriod) Debug() string {
	return fmt.Sprintf(
		"end:\t'%v'\nstart:\t'%v'\nproject:\t'%v'\nnote:\t'%v'\n",
		p.End, p.Start, p.Project, p.Note)
}

func mustOpenDB() *sql.DB {
	envVar := "$PUNCH_CARD"
	dbPath := os.ExpandEnv(envVar)
	if len(dbPath) < 1 {
		fmt.Fprintf(
			os.Stderr,
			"Error: %s not defined, expected as path to database\n",
			envVar)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Failed to open database at path:\n\t%s\n%s\n",
			dbPath, err)
		os.Exit(1)
	}
	return db
}

func mustQueryMostRecent(db *sql.DB, project string) []payPeriod {
	rows, err := db.Query(`
		SELECT * FROM paychecks
		WHERE project=?
		ORDER BY endclusive DESC
		LIMIT 1
	`, project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to query for project '%s': %s\n", project, err)
		os.Exit(1)
	}
	defer rows.Close()

	var periods []payPeriod
	for rows.Next() {
		p := payPeriod{}
		err := rows.Scan(&p.End, &p.Start, &p.Project, &p.Note)
		if err == nil {
			periods = append(periods, p)
		} else {
			fmt.Fprintf(os.Stderr, "Failed reading record: %v\n", err)
		}
	}

	if err := rows.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Query error: %v\n", err)
		os.Exit(1)
	}

	return periods
}

func usage() {
	fmt.Fprintf(os.Stderr, `
Usage: -clientopt CLIENT
	Runs common queries on paycheck history database, and parses the results.

Current -clientopt flags accepting a CLIENT argument:
`)
	flag.PrintDefaults()
}

func main() {
	lastfor := flag.String("lastfor", "",
		"Fetch the last paycheck date for a client")
	flag.Usage = usage
	flag.Parse()
	if len(*lastfor) > 0 {
		db := mustOpenDB()

		mostRecent := mustQueryMostRecent(db, *lastfor)
		if len(mostRecent) > 0 {
			fmt.Printf("%d\n", mostRecent[0].End)
		} else {
			fmt.Fprintf(os.Stderr, "No records for %s\n", *lastfor)
		}
		os.Exit(0)
	}

	usage()
	os.Exit(1)
}
