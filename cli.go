package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var helpRegexp *regexp.Regexp = regexp.MustCompile("(\b|^)(help|h)(\b|$)")
var helpLongRegexp *regexp.Regexp = regexp.MustCompile("(\b|^)(help)(\b|$)")

func isDbReadableFile() (string, os.FileInfo, error) {
	p := os.Getenv(dbEnvVar)
	if len(p) == 0 {
		return "", nil, fmt.Errorf("$%s is not set", dbEnvVar)
	}

	f, e := os.Stat(p)
	if e != nil {
		return p, f, fmt.Errorf(
			"$%s could not be read; tried, '%s'", dbEnvVar, p)
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
	helpDoc := helpCli()
	if helpLongRegexp.MatchString(firstArgChars) {
		helpDoc = helpManual()
		if len(os.Args) > 2 {
			secondArg := strings.TrimSpace(os.Args[2])
			if isSubCmd(secondArg) {
				switch secondArg {
				case "p", "punch":
					helpDoc = helpCmdPunch(false /*cliOnly*/)
				case "bill":
					helpDoc = helpCmdBill(false /*cliOnly*/)
				case "q", "query":
					helpDoc = helpCmdQuery(false /*cliOnly*/)
				}
			}
		}
	}
	fmt.Fprint(os.Stderr, helpDoc)
	os.Exit(0)
}

// TODO(zacsh) allow for global flag to indicate punch in/out renderings should
// be in their original unix timestamp (rather than time.Unix().String()
// rendering)
func main() {
	if len(os.Args) > 1 {
		maybeHandleHelpCli()
	}


	// TODO(zacsh) finish create.go for graceful first-time creation, eg:
	//   https://github.com/jzacsh/punch/blob/a1e40862a7203613cd/bin/punch#L240-L241
	dbPath, dbInfo, e := isDbReadableFile()
	if e != nil {
		fmt.Fprintf(os.Stderr, "Error checking database (see -h): %s\n", e)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		if e := cardQuery(dbInfo, dbPath, []string{queryDefaultCmd}); e != nil {
			fmt.Fprintf(os.Stderr, "status check: %s\n")
			os.Exit(1)
		}
		return
	}

	switch os.Args[1] {
	case "p", "punch":
		if e := processPunch(dbPath, os.Args[2:]); e != nil {
			fmt.Fprintf(os.Stderr, "punch failed: %s\n", e)
			os.Exit(1)
		}
	case "bill":
		if e := markPayPeriod(dbPath, os.Args[2:]); e != nil {
			fmt.Fprintf(os.Stderr, "bill failed: %s\n", e)
			os.Exit(1)
		}
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
