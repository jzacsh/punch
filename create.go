package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

/**
TODO(zacsh) write code to create a punchcard from scratch

 CREATE TABLE punchcard (
     punch       INTEGER NOT NULL PRIMARY KEY,
     status      INTEGER NOT NULL,
     project     TEXT NOT NULL,
     note        TEXT
 );

 CREATE TABLE paychecks (
     endclusive   INTEGER NOT NULL PRIMARY KEY,
     startclusive INTEGER NOT NULL,
     project      TEXT NOT NULL,
     note         TEXT
 );
*/

// TODO(zacsh) finish create.go for graceful first-time creation, eg:
//   https://github.com/jzacsh/punch/blob/a1e40862a7203613cd/bin/punch#L240-L241
func subCmdCreate(dbPath string) error {
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

	fmt.Fprint(os.Stderr, "[dbg] not yet implemented, about to auto-create...\n") // TOOD remove
	return nil
}
