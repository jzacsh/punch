package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

func scanToCard(rows *sql.Rows) (*CardSchema, error) {
	raw := &CardSchemaRaw{}
	if e := rows.Scan(&raw.Punch, &raw.Status, &raw.Project, &raw.Note); e != nil {
		return nil, e
	}
	return raw.toCard(), nil
}

func dump(db *sql.DB) error {
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
			fromStatus(punch.Status),
			punch.Project,
			fromNote(punch.Note))
	}

	return nil
}

func cardQuery(dbInfo os.FileInfo, dbPath string, args []string) error {
	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return errors.New(fmt.Sprintf("punch cards: %s", e))
	}

	if len(args) == 0 {
		return dump(db)
	}

	return nil
}
