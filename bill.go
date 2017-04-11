package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func markPayPeriod(dbPath string, args []string) error {
	if len(args) < 3 {
		return errors.New(fmt.Sprintf(
			"expect at least 3: CLIENT FROM TO [-n NOTE] but only got %d args", len(args)))
	}

	client := strings.TrimSpace(args[0])
	fromStamp := strings.TrimSpace(args[1])
	toStamp := strings.TrimSpace(args[2])
	var note string
	if len(args) > 3 {
		note = strings.TrimSpace(strings.Join(args[3:], " "))
	}

	fmt.Fprintf(os.Stderr, "[dbg] CLIENT='%s' from='%s', to='%s', note='%s'\n",
		client, fromStamp, toStamp, note) // TODO remove

	return /*nil*/ errors.New("not yet implemented")
}
