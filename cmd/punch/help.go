package main

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"os/exec"
	"strings"
)

var VersionUrl, VersionRef, VersionDate string

const AppName string = "punch"

// NOTE: all above defined at build time

const dbEnvVar string = "PUNCH_CARD"

const queryDefaultCmd string = "status"

const helpCliPattern string = "punch [punch|bill|query|delete|amend|seek] [...]"
const helpDoesWhat string = "Logs & reports time worked on any project"

func isSubCmd(str string) bool {
	return str == "p" || str == "punch" ||
		str == "bill" ||
		str == "q" || str == "query" ||
		str == "d" || str == "delete" ||
		str == "a" || str == "amend" ||
		str == "s" || str == "seek"
}

// Name, synopsis, description
func helpSectionHeader() string {
	return fmt.Sprintf(`NAME
  %s - %s

SYNOPSIS
  %s

DESCRIPTION
  Manages your work clock, allowing you to "punch in" or "punch out" and query
  for some obvious stats & reporting you might want.
`, AppName, helpDoesWhat, helpCliPattern)
}

func helpCmdPunch(cliOnly bool) string {
	var punchHelp string
	if !cliOnly {
		punchHelp = `
    Allows punching in & out of work on a "client"/"project" indiciated by a
    CLIENT string (an alphanumeric string of characters).

    There are two uses for this command:
    - 1) starting work: "punching in" to start the clock for some project or client
    - 2) stopping work: "punching out" to stop the clock for some project or client
    In both cases CLIENT indicates which client/project to starting/stopping work on.

    If CLIENT Is not provided, it's assumed you're trying to implicitly punch
    out. If you're not punched into exactly one CLIENT already, then no-arg punching
    will have no safe assumptions to make, and the command will fail.

    Optionally, passing -n NOTE indicates that NOTE string should be stored for
    future reference for this punchcard entry.`
	}
	return fmt.Sprintf("  p|punch    [CLIENT] [-n NOTE]\n%s\n", punchHelp)
}

func helpCmdBill(cliOnly bool) string {
	var billHelp string
	if !cliOnly {
		billHelp = `
    Records durations of time over which a payperiod occurs. To see its impact,
    as a dry run, pass -d. Duration of the pay period is defined to be the
    inclusive span between the unix time stamps FROM and TO.

    See DATE(1) under EXAMPLES for more on TO/FROM timestamps.

    If a TO stamp is not provided, this implies the duration should end at the
    most recent punch-out. Useful if you've just punched-out to mark the end of
    a billable project.

    If a FROM stamp is not provided, this implies the beginning of the duration
    should be:
     1) the TO-duration of the previous recorded payperiod
     2) if no previous payperiod is found, the earliest punch stamp under CLIENT
        in the punchcard table.

    Note: data on billing is not in anyway related to the data kept on punches.
    When "query bills" reports time worked over a pay period, it merely
    correlates overlaps in duration indicated by the payperiod with any
    durations logged through punches.`
	}
	return fmt.Sprintf(
		"  bill CLIENT [-d] [-f FROM] [-t TO] [-n NOTE]\n%s\n",
		billHelp)
}

func helpCmdDelete(cliOnly bool) string {
	var deleteHelp string
	if !cliOnly {
		deleteHelp = `
    Interactively deletes payperiods or punches. The two cases are described
    below. The -d flag indicates this is a dry-run, and no modifications should
    be made.

    Case 1: If 'bill' argument is passed, then a CLIENT's payperiod is deleted
    where AT matches the payperiod's FROM timestamp.

    Case 2: If 'punch' is specified then one of two things is done for CLIENT's
    punches:
     i) If AT matches a punch-out, and there are no punches since AT, then that
        punch-out is deleted (ie: punch session is extended to put you back on
        the clock).
    ii) If AT matches a punch-in, then the entire session is deleted (from
        punch-in to its corresponding punch-out, if one exists)`
	}
	return fmt.Sprintf(
		"  d|delete bill|punch CLIENT [-d] AT\n%s\n",
		deleteHelp)
}

func helpCmdQuery(cliOnly bool) string {
	var queryHelp string
	if !cliOnly {
		queryHelp = `
    Allows you to query your work activity, where QUERY is any one of the
    below. If no QUERY is provided, 'dump' is assumed.
  - list: Lists all "clients"/"projects" for which records currently exist
  - dump: pseudo CSV-esque dump of database values, ordered by punch-date,
    one-punch per-line.
  - report CLIENT [FROM_STAMP]: Prints a general report on the CLIENT provided.
    If a unix timestamp FROM_STAMP (in seconds) is specified, it's used as
    furthest boundary back to fetch records. See DATE(1) under EXAMPLES for more
    on timestamps.
  - status: prints running-time on any currently punched-into projects.
  - bills [-last] [CLIENT ...]: prints report of payperiod under all CLIENT names.
    If CLIENT is not provided, prints report consecutively for each CLIENT
    returned by "query list"
    If -last is provided, prints the scripting-friendly end-timestamp (and its
    human-readable rendering) of the most recent payperiod found for CLIENT.
    This option requires that exactly one CLIENT be provided.`
	}
	return fmt.Sprintf("  q|query    [QUERY...]\n%s\n", queryHelp)
}

func helpCmdAmend(cliOnly bool) string {
	var amendHelp string
	if !cliOnly {
		amendHelp = `
    Allows replacing (or deleting) notes previously set to the punch record
    whose timestamp matches TARGET_STAMP exactly. Note for matching punch is
    replaced with NOTE.

    If NOTE is not provided, the note for said punch is deleted. See DATE(1)
    under EXAMPLES for more on TO/FROM timestamps.`
	}
	return fmt.Sprintf("  a|amend    TARGET_STAMP [NOTE]\n%s\n", amendHelp)
}

func helpCmdSeek(cliOnly bool) string {
	var seekHelp string
	if !cliOnly {
		seekHelp = `
    Allows changing a faulty punch-out stamp, FAULTY_STAMP, to the timestamp
    SEEK_TO.

    Passing -c indicates SEEK_TO is Closing a still-open session whose punch-in
    is the timestamp STILL_OPEN.

    If -d is passed, "dry-run", no changes will be made.`
	}
	return fmt.Sprintf("  s|seek  [-d] SEEK_TO  FAULTY_STAMP | -c STILL_OPEN\n%s\n", seekHelp)
}

// Subcommands
func helpSectionCommands() string {
	return fmt.Sprintf(`COMMANDS
  One of the below sub-commands is expected, otherwise "query %s" is assumed.

  h|help [COMMAND]

    Prints help documentation just for one of the below commands per COMMAND.
    Otherwise prints all documentation. All of -h, --h, h just print a brief CLI
    pseudo-grammar doc.

%s
%s
%s
%s
%s
%s`, queryDefaultCmd,
		helpCmdPunch(false /*cliOnly*/),
		helpCmdBill(false /*cliOnly*/),
		helpCmdDelete(false /*cliOnly*/),
		helpCmdQuery(false /*cliOnly*/),
		helpCmdAmend(false /*cliOnly*/),
		helpCmdSeek(false /*cliOnly*/))
}

// Environment & Examples
func helpSectionFooter() string {
	var buildInfo string
	if len(VersionDate) < 1 || len(VersionRef) < 1 || len(VersionUrl) < 1 {
		buildInfo = "This binary not built with its usual tool, build data unavailable."
	} else {
		buildInfo = fmt.Sprintf(`This binary built %s at git ref %s.

  To see the source for this program and a full copy of its license, see:
    %s
		`, VersionDate, VersionRef, VersionUrl)
	}

	return fmt.Sprintf(`ENVIRONMENT
  Work clock is an SQLite3 database file path, which is expected to be in $%s
  environment variable.

EXAMPLES
  Common 'punch' command lines:
   $ punch # same as "punch query %s"
   ch: 99:01 so far
   $ punch p -n 'phew, finished proving hypothesis'
   $ punch
   Not on the clock.
   $ punch p puzzles -n 'free time to tackle tetris in Brainfuck'

  Unix timestamps can be easily obtained. Passing '+%%s' to GNU's DATE(1)
  utilizes its excellent parser output to produce valid unix timestamps:
   $ date    # Tue Apr 11 08:58:26 EDT 2017
   $ date     --date=yesterday
   Mon Apr 10 08:58:23 EDT 2017
   $ date     --date="8pm next Fri"
   Fri Apr 14 20:00:00 EDT 2017
   $ date +%%s --date="8pm next Fri"
   1492214400 # perfect unix timestamp in seconds

BUILD INFORMATION
  %s
`, dbEnvVar, queryDefaultCmd, buildInfo)
}

func helpManual() string {
	return fmt.Sprintf(
		"%s\n%s\n%s\n",
		helpSectionHeader(),
		helpSectionCommands(),
		helpSectionFooter())
}

// Either pipes payload to a pager, or straight to stdout
// Mostly taken from http://stackoverflow.com/a/21739281/287374
func maybePipeToPager(payload string) {
	pagerEnv := os.Getenv("PAGER")
	if !terminal.IsTerminal(int(os.Stdout.Fd())) || len(pagerEnv) < 1 {
		fmt.Print(payload)
		return
	}
	pipeOut, pipeIn := io.Pipe()

	pagerCmd := exec.Command(pagerEnv)
	pagerCmd.Stdin = pipeOut
	pagerCmd.Stdout = os.Stdout
	pagerCmd.Stderr = os.Stderr

	childChan := make(chan struct{})
	go func() {
		defer close(childChan)
		if e := pagerCmd.Run(); e != nil {
			fmt.Print(payload)
			fmt.Fprintf(os.Stderr,
				"WARNING: dumped direct to stdout, as $PAGER(%s) failed with:\n%s\n",
				pagerEnv, e)
			os.Exit(0)
		}
	}()

	fmt.Fprint(pipeIn, payload)
	pipeIn.Close()

	<-childChan
}

// the tl;dr version of helpManual
func helpCli() string {
	return fmt.Sprintf("usage: %s\n  %s%s\n\n%s%s%s%s%sSee --help for more\n",
		helpCliPattern,
		helpDoesWhat,
		helpCmdPunch(true /*cliOnly*/),
		helpCmdBill(true /*cliOnly*/),
		helpCmdDelete(true /*cliOnly*/),
		helpCmdQuery(true /*cliOnly*/),
		helpCmdAmend(true /*cliOnly*/),
		helpCmdSeek(true /*cliOnly*/))
}

func subCmdHelp(firstArgChars string, args []string) {
	helpDoc := helpCli()
	if helpLongRegexp.MatchString(firstArgChars) {
		helpDoc = helpManual()
		if len(args) > 1 {
			secondArg := strings.TrimSpace(args[1])
			if isSubCmd(secondArg) {
				switch secondArg {
				case "p", "punch":
					helpDoc = helpCmdPunch(false /*cliOnly*/)
				case "bill":
					helpDoc = helpCmdBill(false /*cliOnly*/)
				case "q", "query":
					helpDoc = helpCmdQuery(false /*cliOnly*/)
				case "d", "delete":
					helpDoc = helpCmdDelete(false /*cliOnly*/)
				case "a", "amend":
					helpDoc = helpCmdAmend(false /*cliOnly*/)
				case "s", "seek":
					helpDoc = helpCmdSeek(false /*cliOnly*/)
				}
				helpDoc += "\n  See --help without arguments to see full doc.\n"
			}
		}
	}
	maybePipeToPager(helpDoc)
}
