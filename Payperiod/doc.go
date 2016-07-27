// Quick hack, written in go, to query the latest from an sqlite3 table whose
// schema is:
//    CREATE TABLE paychecks (
//      endclusive   INTEGER NOT NULL PRIMARY KEY,
//      startclusive INTEGER NOT NULL,
//      project      TEXT NOT NULL,
//      note         TEXT
//    );
//
// Such that:
// -  "project" values are the primary keys of the punch database
// -  "note" are free-form, human-readable strings, not intended for parsing
// -  "endclusive" is the unix timestamp of the end of the last payperiod (ie: `date +%s`)
// -  "startclusive" is the unix timestamp of the start of the last payperiod
//    (inclusive, just like values in "endclusive" column)
//
// Currently run as:
//  $ go run Payperiod/main.go -lastfor "myclient"
//
// Particularly useful for i3blocks "cummulative" for a given period; see:
//   https://github.com/jzacsh/dotfiles/blob/5806a63d0aff0/.config/i3blocks/punchmon

package main
