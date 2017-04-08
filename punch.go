package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const dbEnvVar string = "PUNCH_CARD"
const usageDoc string = `usage:   punch in|out|query

DESCRIPTION
  Manages your work clock, allowing you to "punch in" or "punch out" and query
  for some obvious stats & reporting you might want.

  Work clock is an SQLite3 database file, path to which is expected to be in
  $%s environment variable

COMMANDS
  i|in    CLIENT [NOTE]
    Allows you to punch into work on a "client" or "project" (how exactly you
    classify your work with this time keeping program is irrelevant to the
    program).
  - CLIENT: Name of the client to punch into work for
  - NOTE: Notes you'd like to show up when reporting, anything you want to be
    on the record about this work period. (Eg: "trying to finish design doc v3").


  o|out    [CLIENT] [-n NOTE]
    Allows you to punch out of work on which ever
  - CLIENT: Required if you're currently clocked into multiple clients (Eg: if
    perhaps you're using "clients" to mean "projects"). Defaults to the
    currently clocked-in client, if only one. Causes an error if you're not clocked
     into anything.
  - NOTE: Optional note, identical in purpose to that of 'in' command's NOTE option.

  q|query    [CLIENT]
    Allows you to query your work activity.
`

var helpRegexp *regexp.Regexp = regexp.MustCompile("(\b|^)(help|h)(\b|$)")

func failNotYetImplemented(whatFailed string) {
	fmt.Fprintf(
		os.Stderr, "nothing implemented yet (not even '%s' subcommand)\n",
		whatFailed)
	os.Exit(99)
}

func main() {
	if len(os.Args) < 2 ||
		helpRegexp.MatchString(strings.Replace(os.Args[1], "-", "", -1)) {
		fmt.Fprintf(os.Stderr, usageDoc, dbEnvVar)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "i", "in":
		failNotYetImplemented(os.Args[1])
	case "o", "out":
		failNotYetImplemented(os.Args[1])
	case "q", "query":
		failNotYetImplemented(os.Args[1])
	default:
		fmt.Fprintf(os.Stderr,
			"valid sub-command required (ie: not '%s'); try --h for usage\n", os.Args[1])
		os.Exit(1)
	}
}
