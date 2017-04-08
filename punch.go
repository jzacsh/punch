package main

import (
	"errors"
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

  q|query    [QUERY...]
    Allows you to query your work activity, where QUERY is any one of the
    below. If no QUERY is provided, a dump of the database as comma-separated
    values will be generated (ordered by punch-date, one-punch per-line).
  - list: Lists all "clients"/"projects" for which records currently exist
`

var helpRegexp *regexp.Regexp = regexp.MustCompile("(\b|^)(help|h)(\b|$)")

func failNotYetImplemented(whatFailed string) {
	fmt.Fprintf(
		os.Stderr, "nothing implemented yet (not even '%s' subcommand)\n",
		whatFailed)
	os.Exit(99)
}

func isDbReadableFile() (string, os.FileInfo, error) {
	p := os.Getenv(dbEnvVar)
	if len(p) == 0 {
		return "", nil, errors.New(fmt.Sprintf("$%s is not set", dbEnvVar))
	}

	f, e := os.Stat(p)
	if e != nil {
		return p, f, errors.New(fmt.Sprintf(
			"$%s could not be read; tried, '%s'", dbEnvVar, p))
	}

	if f.IsDir() {
		return "", f, errors.New(fmt.Sprintf("$%s must be a regular file", dbEnvVar))
	}
	return p, f, nil
}

// TODO(zacsh) allow for global flag to indicate punch in/out renderings should
// be in their original unix timestamp (rather than time.Unix().String()
// rendering)
func main() {
	if len(os.Args) < 2 ||
		helpRegexp.MatchString(strings.Replace(os.Args[1], "-", "", -1)) {
		fmt.Fprintf(os.Stderr, usageDoc, dbEnvVar)
		os.Exit(1)
	}

	// TODO(zacsh) graceful first-time creation, eg:
	//   https://github.com/jzacsh/punch/blob/a1e40862a7203613cd/bin/punch#L240-L241
	dbPath, dbInfo, e := isDbReadableFile()
	if e != nil {
		fmt.Fprintf(os.Stderr, "Error checking database (see -h): %s\n", e)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "i", "in":
		failNotYetImplemented(os.Args[1])
	case "o", "out":
		failNotYetImplemented(os.Args[1])
	case "q", "query":
		if e := cardQuery(dbInfo, dbPath, os.Args[2:]); e != nil {
			fmt.Fprintf(os.Stderr, "query failed: %s\n", e)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr,
			"valid sub-command required (ie: not '%s'); try --h for usage\n", os.Args[1])
		os.Exit(1)
	}
}
