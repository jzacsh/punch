package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"strconv"
	"strings"
	"time"
)

type DeleteCmd struct {
	Action   string
	Client   string
	IsDryRun bool
	At       time.Time
}

func (d *DeleteCmd) String() string {
	return fmt.Sprintf(
		"Delete %s for '%s' at %s (timestamp %d) [dry-run=%t]",
		d.Action,
		d.Client,
		d.At.Format(format_dateTime),
		d.At.Unix(),
		d.IsDryRun)
}

func parseDeleteCmd(args []string) (*DeleteCmd, error) {
	cmd := &DeleteCmd{}
	if len(args) < 3 {
		return cmd, fmt.Errorf(
			"expected at least 3 args per 'bill|punch CLIENT [-d] AT', got %d",
			len(args))
	}
	cmd.Action = strings.TrimSpace(args[0])
	if cmd.Action != "bill" && cmd.Action != "punch" {
		return cmd, fmt.Errorf(
			"expected either 'bill' or 'punch', got '%s'", cmd.Action)
	}

	cmd.Client = strings.TrimSpace(args[1])
	if !isValidClient(cmd.Client) {
		return cmd, fmt.Errorf("invalid CLIENT, '%s'", cmd.Client)
	}

	atCmd := args[2]
	cmd.IsDryRun = false
	if len(args) > 3 {
		if strings.TrimSpace(args[2]) != "-d" {
			return cmd, fmt.Errorf("unrecognized cmd at '%s'",
				strings.TrimSpace(strings.Join(args[2:], " ")))
		}
		cmd.IsDryRun = true
		atCmd = args[3]
	}

	atStamp, e := strconv.ParseInt(strings.TrimSpace(atCmd), 10, 64)
	if e != nil {
		return cmd, fmt.Errorf("parsing AT unix timestamp, '%s': %s", atCmd, e)
	}
	cmd.At = time.Unix(atStamp, 0 /*nanoseconds*/)
	return cmd, nil
}

func subCmdDelete(dbPath string, args []string) error {
	cmd, e := parseDeleteCmd(args)
	if e != nil {
		return fmt.Errorf("parsing command: %s", e)
	}

	db, e := sql.Open("sqlite3", dbPath)
	if e != nil {
		return fmt.Errorf("delete from db: %s", e)
	}
	defer db.Close()

	fmt.Printf("%s\n", cmd)

	var stmt *sql.Stmt
	if cmd.Action == "bill" {
		stmt, e = db.Prepare(`
		 -- TODO something w/ 2 questions
		`)
		if e != nil {
			return fmt.Errorf("preparing SQL for deletion: %s", e)
		}
	} else {
		return fmt.Errorf("%s deletion not yet implemented :(", cmd.Action) // TODO
	}

	if cmd.IsDryRun {
		fmt.Fprint(os.Stderr, "[-d]ry-run: finishing early; NO changes written\n")
		return nil
	}

	if cmd.Action == "bill" {
		stmt.Exec(cmd.Client, cmd.At.Unix())
	} else {
		return fmt.Errorf(
			"%s deletion exec not yet implemented :(",
			cmd.Action) // TODO
	}

	return fmt.Errorf("not yet implemented :(") // TODO
}
