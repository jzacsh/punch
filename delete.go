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
		"Delete '%s'-%s at %s [@%d] [dry-run=%t]",
		d.Client,
		d.Target,
		d.At.Format(format_dateTime),
		d.At.Unix(),
		d.IsDryRun)
}

// `punchOut` will be -1 when d.Target is 'bill', else:
// - punchOut of -1 indicates `d` is a request to delete a punch-out, meaning
//   there's no corresponding punch-out because `d` itself represents a punch-out.
// - punchOut of > -1 indicates `d` is a punch-in, and punchOut is the timestamp
//   of `d`'s corresponding punch-out record.
func (d *DeleteCmd) Report(db *sql.DB) (punchOut int64, _ error) {
	fmt.Printf("%s...\n", d)
	punchOut = -1

	if d.isTargetingBill() {
		rows, e := db.Query(`
		SELECT * FROM paychecks
		WHERE project IS ?
		AND startclusive IS ?
		;`, d.Client, d.At.Unix())
		if e != nil {
			return punchOut, fmt.Errorf("querying DB: %s", e)
		}
		defer rows.Close()
		foundTarget := false
		for rows.Next() {
			b, e := scanToBill(rows)
			if e != nil {
				return punchOut, fmt.Errorf("parsing DB response: %s", e)
			}

			if foundTarget {
				return punchOut, fmt.Errorf("malformed data: found TWO payperiods sharing start time")
			}

			foundTarget = true
			fmt.Printf(
				"FOUND target bill to delete [%s]:\n%s\n",
				getTZContext(),
				b.String(false /*showTimezone*/))
		}
		if !foundTarget {
			return punchOut, fmt.Errorf(
				"no '%s' payperiods start at %s",
				d.Client, d.At.Format(format_dateTime))
		}
	} else {
		rows, e := db.Query(`
		SELECT * FROM punchcard
		WHERE project IS ?
		AND punch >= ?
		ORDER BY punch ASC
		LIMIT 2
		;`, d.Client, d.At.Unix())
		if e != nil {
			return punchOut, fmt.Errorf("querying DB: %s", e)
		}
		defer rows.Close()

		i := 0
		var match, second *CardSchema
		for rows.Next() {
			c, e := scanToCard(rows)
			if e != nil {
				return punchOut, fmt.Errorf("parsing DB response: %s", e)
			}

			if i == 0 {
				if c.Punch != d.At {
					return punchOut, fmt.Errorf("no '%s' punch at %s",
						d.Client, d.At.Format(format_dateTime))
				}
				match = c
			} else {
				second = c
			}

			i++
		}
		if match == nil {
			return punchOut, fmt.Errorf(
				"no '%s' punches found between %s and now",
				d.Client,
				d.At.Format(format_dateTime))
		}

		isSessionDeletion := match.IsStart // Deleting an entire session
		if isSessionDeletion {
			if second.IsStart {
				return punchOut, fmt.Errorf(
					"malformed db: found TWO punch-ins in a row, second at %d",
					second.Punch.Unix())
			}
			if second == nil {
				fmt.Printf(
					"Effectively deletes an active %s-session that started %s ago\n\tnote: '%s'\n",
					d.Client,
					time.Now().Sub(d.At),
					fromNote(match.Note))
			} else {
				punchOut = second.Punch.Unix()

				session := fmt.Sprintf(
					"\tstart note: '%s'\n\tend   note: '%s'",
					fromNote(match.Note), fromNote(second.Note))
				fmt.Printf(
					"Effectively deletes entire %s-session that ended %s [@%d]:\n%s\n",
					second.Punch.Sub(d.At),
					second.Punch.Format(format_dateTime),
					punchOut,
					session)
			}
		} else {
			// We're effectively punching back in (ie: we punched out out of a
			// work-session, and want to undo that, indicating we've still been
			// working until now, this whole time).

			rows, e := db.Query(`
			SELECT COUNT(DISTINCT punch)
			FROM punchcard
			WHERE project IS ?
			AND punch > ?
			;`, d.Client, d.At.Unix())
			if e != nil {
				return punchOut, fmt.Errorf("querying DB: %s", e)
			}
			defer rows.Close()

			count := -1
			for rows.Next() {
				if e := rows.Scan(&count); e != nil {
					return punchOut, fmt.Errorf("querying DB: %s", e)
				}
			}
			if count == -1 {
				panic("somehow pointer not populated by successful rows.Scan()")
			}

			if count != 0 {
				return punchOut, fmt.Errorf(
					"re-opening work session with new sessions opened since will cause data inconsistency (%d punches found since). HINT: to delete an ENTIRE session, delete its punch-IN time.", count)
			}

			fmt.Printf(
				"Effectively re-opening session that ended %s ago at %s\n",
				time.Now().Sub(d.At),
				d.At.Format(format_dateTime))
		}
	}

	return punchOut, nil
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

	punchOut, e := cmd.Report(db)
	if e != nil {
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
		if punchOut == -1 { // cmd.At is a punch-out, we want to re-open the session
			stmt, e = db.Prepare(`
			DELETE FROM punchcard
			WHERE project iS ?
			AND punch IS ?
			;`)
			if e != nil {
				return fmt.Errorf("preparing SQL for deletion: %s", e)
			}
		} else { // cmd.At is a punch-in, we want to delete the whole session
			stmt, e = db.Prepare(`
			DELETE FROM punchcard
			WHERE project iS ?
			AND punch IN (?, ?)
			;`)
			if e != nil {
				return fmt.Errorf("preparing SQL for deletion: %s", e)
			}
		}
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
		if punchOut == -1 {
			if _, e := stmt.Exec(cmd.Client, cmd.At.Unix()); e != nil {
				return e
			}
		} else {
			if _, e := stmt.Exec(cmd.Client, cmd.At.Unix(), punchOut); e != nil {
				return e
			}
		}
	}

	fmt.Println("Done.")
	return nil
}
