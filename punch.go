package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strings"
)

func parseArgs(args []string) (string, string, error) {
	if len(args) == 0 {
		return "", "", nil
	}
	var client, note, noteRaw string
	clientOrFlag := strings.TrimSpace(args[0])
	if len(args) == 1 {
		if clientOrFlag == "-n" {
			return "", "", fmt.Errorf(
				"expected CLIENT, -n NOTE, or CLIENT -n NOTE, but got just -n")
		}

		client = clientOrFlag
		if len(client) == 0 {
			return "", "", fmt.Errorf(
				"CLIENT must be non-empty (or -n provided), but got '%s'", args[0])
		}
		return client, note, nil
	} else {
		flagOrNoteChunk := strings.TrimSpace(args[1])
		if clientOrFlag == "-n" {
			noteRaw = strings.Join(args[1:], " ")
		} else {
			client = clientOrFlag
			noteRaw = strings.Join(args[2:], " ")

			if flagOrNoteChunk != "-n" {
				return "", "", fmt.Errorf(
					"expected CLIENT [-n NOTE], but got CLIENT='%s' followed by, '%s'",
					clientOrFlag, noteRaw)
			}
		}
	}

	note = strings.TrimSpace(noteRaw)
	if len(note) < 1 {
		return "", "", fmt.Errorf(
			"expected -n NOTE but '-n %s'",
			noteRaw)
	}

	return client, note, nil
}

func getImpliedClient(db *sql.DB) (string, error) {
	rows, e := db.Query(`
		SELECT * FROM punchcard
		GROUP BY project
		ORDER BY punch DESC;
	`)
	if e != nil {
		return "", e
	}
	defer rows.Close()

	var punchedInto string
	for rows.Next() {
		card, e := scanToCard(rows)
		if e != nil {
			return "", fmt.Errorf("punch cards: %s", e)
		}

		if card.IsStart {
			if len(punchedInto) > 0 {
				return "", fmt.Errorf(
					"implying one CLIENT is on clock, but found 2: '%s' & '%s'",
					punchedInto, card.Project)
			}
			punchedInto = card.Project
		}
	}

	if len(punchedInto) == 0 {
		return "", errors.New("implying one CLIENT is on clock, but none are")
	}

	return punchedInto, nil
}

func isPunchIn(db *sql.DB, client string, isImplicitPunchOut bool) (bool, error) {
	if isImplicitPunchOut {
		return false, nil
	}

	rows, e := db.Query(`
		SELECT * FROM punchcard
		WHERE project IS ?
		ORDER BY punch DESC
		LIMIT 1;
	`, client)
	if e != nil {
		return false, e
	}
	defer rows.Close()

	for rows.Next() {
		card, e := scanToCard(rows)
		if e != nil {
			return false, e
		}
		return !card.IsStart, nil
	}
	return false, nil
}

func punchProject(db *sql.DB, card *CardSchemaSQL) error {
	stmt, e := db.Prepare(`
		INSERT INTO
		punchcard(punch, status, project, note)
		VALUES (?, ?, ?, ?)
	`)
	if e != nil {
		return e
	}

	_, e = stmt.Exec(card.Punch, card.Status, card.Project, card.Note)
	// TODO(zacsh) expose result val here via debug flags on cli

	return e
}

func subCmdPunch(dbPath string, args []string) error {
	explicitClient, note, e := parseArgs(args)
	if e != nil {
		return e
	}

	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return fmt.Errorf("punch cards: %s", e)
	}
	defer db.Close()

	isImplicitPunchOut := false
	client := explicitClient
	if len(client) == 0 {
		isImplicitPunchOut = true
		client, e = getImpliedClient(db)
		if e != nil {
			return e
		}
	}

	isPunchIn, e := isPunchIn(db, client, isImplicitPunchOut)
	if e != nil {
		return e
	}

	sqlCard := buildCardSQL(isPunchIn, client, note)
	return punchProject(db, sqlCard)
}
