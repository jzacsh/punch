package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var helpRegexp *regexp.Regexp = regexp.MustCompile("(\b|^)(help|h)(\b|$)")
var helpLongRegexp *regexp.Regexp = regexp.MustCompile("(\b|^)(help)(\b|$)")

// Guarantees env. var value WILL return, even on error not nil
func isDbReadableNonemptyFile() (string, os.FileInfo, error) {
	p := os.Getenv(dbEnvVar)
	if len(p) == 0 {
		return "", nil, fmt.Errorf("$%s is not set", dbEnvVar)
	}

	f, e := os.Stat(p)
	if e != nil {
		return p, f, fmt.Errorf(
			"$%s could not be read; tried, '%s'", dbEnvVar, p)
	}

	if f.Size() < 1 {
		return p, f, fmt.Errorf("$%s is empty file", dbEnvVar)
	}

	if f.IsDir() {
		return "", f, fmt.Errorf("$%s must be a regular file", dbEnvVar)
	}
	return p, f, nil
}

func maybeHandleHelpCli() {
	firstArgChars := strings.Replace(os.Args[1], "-", "", -1)
	if !helpRegexp.MatchString(firstArgChars) {
		return
	}
	subCmdHelp(firstArgChars, os.Args[1:])
	os.Exit(0)
}

// TODO(zacsh) allow for global flag to indicate punch in/out renderings should
// be in their original unix timestamp (rather than time.Unix().String()
// rendering)
func main() {
	if len(os.Args) > 1 {
		maybeHandleHelpCli()
	}

	isCmdDefault := len(os.Args) < 2

	// TODO(zacsh) nit: consider deleting `dbInfo` codepaths
	dbPath, dbInfo, e := isDbReadableNonemptyFile()
	if e != nil {
		if isCmdDefault && len(dbPath) > 0 {
			exitCode := 0
			if e := subCmdCreate(dbPath); e != nil {
				fmt.Fprintf(os.Stderr, "need sqlite3 db: %s\n", e)
				exitCode = 1
			}
			os.Exit(exitCode)
		} else {
			fmt.Fprintf(os.Stderr, "Error checking database (see -h): %s\n", e)
			os.Exit(1)
		}
	}

	if isCmdDefault {
		if e := subCmdQuery(dbInfo, dbPath, []string{queryDefaultCmd}); e != nil {
			fmt.Fprintf(os.Stderr, "status check: %s\n", e)
			os.Exit(1)
		}
		return
	}

	switch os.Args[1] {
	case "p", "punch":
		if e := subCmdPunch(dbPath, os.Args[2:]); e != nil {
			fmt.Fprintf(os.Stderr, "punch failed: %s\n", e)
			os.Exit(1)
		}
	case "bill":
		if e := subCmdBill(dbPath, os.Args[2:]); e != nil {
			fmt.Fprintf(os.Stderr, "bill failed: %s\n", e)
			os.Exit(1)
		}
	case "q", "query":
		if e := subCmdQuery(dbInfo, dbPath, os.Args[2:]); e != nil {
			fmt.Fprintf(os.Stderr, "query failed: %s\n", e)
			os.Exit(1)
		}
	case "d", "delete":
		if e := subCmdDelete(dbPath, os.Args[2:]); e != nil {
			fmt.Fprintf(os.Stderr, "delete failed: %s\n", e)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr,
			"valid sub-command required (ie: not '%s'); try --h for usage\n", os.Args[1])
		os.Exit(1)
	}
}
