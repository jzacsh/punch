package main

import (
	"fmt"
	"os"
)

const usageDoc string = `usage:   punch in|out|query

DESCRIPTION
  Manages your work clock, allowing you to "punch in" or "punch out" and query
  for some obvious stats & reporting you might want.

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
`

func main() {
	fmt.Fprintf(os.Stderr, usageDoc)
	os.Exit(1)
}
