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
	Target   string // "bill" or "punch"
	Client   string
	IsDryRun bool
	At       time.Time
}

func (d *DeleteCmd) isTargetingPunch() bool {
	return !d.isTargetingBill()
}

func (d *DeleteCmd) isTargetingBill() bool {
	return d.Target == "bill"
}

func (d *DeleteCmd) String() string {
	return fmt.Sprintf(
		"Delete '%s'-%s at %s (timestamp %d) [dry-run=%t]",
		d.Client,
		d.Target,
		d.At.Format(format_dateTime),
		d.At.Unix(),
		d.IsDryRun)
}

func (d *DeleteCmd) Report(db *sql.DB) error {
	fmt.Printf("%s...\n", d)

	if d.isTargetingBill() {
		rows, e := db.Query(`
		SELECT * FROM paychecks
		WHERE project IS ?
		AND startclusive IS ?
		;`, d.Client, d.At.Unix())
		if e != nil {
			return fmt.Errorf("querying DB: %s", e)
		}
		defer rows.Close()
		foundTarget := false
		for rows.Next() {
			b, e := scanToBill(rows)
			if e != nil {
				return fmt.Errorf("parsing DB response: %s", e)
			}

			if foundTarget {
				return fmt.Errorf("malformed data: found TWO payperiods sharing start time")
			}

			foundTarget = true
			fmt.Printf(
				"FOUND target bill to delete [%s]:\n%s\n",
				getTZContext(),
				b.String(false /*showTimezone*/))
		}
		if !foundTarget {
			return fmt.Errorf(
				"no '%s' payperiods start at %s",
				d.Client, d.At.Format(format_dateTime))
		}
	} else {
		return fmt.Errorf("reporting for punch-deletions, not yet implemented")
	}

	return nil
}

func parseDeleteCmd(args []string) (*DeleteCmd, error) {
	cmd := &DeleteCmd{}
	if len(args) < 3 {
		return cmd, fmt.Errorf(
			"expected at least 3 args per 'bill|punch CLIENT [-d] AT', got %d",
			len(args))
	}
	cmd.Target = strings.TrimSpace(args[0])
	if cmd.Target != "bill" && cmd.Target != "punch" {
		return cmd, fmt.Errorf(
			"expected either 'bill' or 'punch', got '%s'", cmd.Target)
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

	if e := cmd.Report(db); e != nil {
		return e
	}

	// Do as much as possible before: committing or bailing(dry-run)

	var stmt *sql.Stmt
	if cmd.isTargetingBill() {
		stmt, e = db.Prepare(`
		DELETE FROM paychecks
		WHERE project iS ?
		AND startclusive IS ?
		;`)
		if e != nil {
			return fmt.Errorf("preparing SQL for deletion: %s", e)
		}
	} else {
		return fmt.Errorf("%s deletion not yet implemented :(", cmd.Target) // TODO
	}

	if cmd.IsDryRun {
		fmt.Fprint(os.Stderr, "[-d]ry-run: finishing early; NO changes written\n")
		return nil
	}

	if cmd.isTargetingBill() {
		if _, e := stmt.Exec(cmd.Client, cmd.At.Unix()); e != nil {
			return e
		}
	} else {
		return fmt.Errorf(
			"%s deletion exec not yet implemented :(",
			cmd.Target) // TODO
	}

	fmt.Println("Done.")
	return nil
}
