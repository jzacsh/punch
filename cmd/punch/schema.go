package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// see: https://marcesher.com/2014/10/13/go-working-effectively-with-database-nulls/
func toNullString(optional string) sql.NullString {
	value := strings.TrimSpace(optional) // pertinent specifically to punch schemas
	return sql.NullString{String: value, Valid: len(value) != 0}
}

// see: https://marcesher.com/2014/10/13/go-working-effectively-with-database-nulls/
func fromNullString(optional sql.NullString) string {
	if !optional.Valid {
		return "" // golang zero-value
	}
	return optional.String
}

type BillSchemaSQL struct {
	Endclusive   int // primary key
	Startclusive int
	Project      string
	Note         sql.NullString
}

func (b *BillSchemaSQL) toBill() *BillSchema {
	return &BillSchema{
		Endclusive:   time.Unix(int64(b.Endclusive), 0 /*nanoseconds*/),
		Startclusive: time.Unix(int64(b.Startclusive), 0 /*nanoseconds*/),
		Project:      b.Project,
		Note:         fromNullString(b.Note),
	}
}

func (b *BillSchema) toSQL() *BillSchemaSQL {
	return &BillSchemaSQL{
		Endclusive:   int(b.Endclusive.Unix()),
		Startclusive: int(b.Startclusive.Unix()),
		Project:      b.Project,
		Note:         toNullString(b.Note),
	}
}

func (b *BillSchema) String(showTimezone bool) string {
	var start, end string

	if showTimezone {
		start = b.Startclusive.Format(format_dateTime)
		end = b.Endclusive.String() // unnecessary twice
	} else {
		start = b.Startclusive.Format(format_dateTime)
		end = b.Endclusive.Format(format_dateTime)
	}

	return fmt.Sprintf("%s, %s, %s, %s",
		b.Project,
		start, end,
		fromNote(b.Note))
}

type BillSchema struct {
	Endclusive   time.Time
	Startclusive time.Time
	Project      string
	Note         string //optional
}

type CardSchemaSQL struct {
	Punch   int // unix stamp seconds; primary key
	Status  int // (pseudo-boolean) 1,0
	Project string
	Note    sql.NullString
}

type CardSchema struct {
	Punch   time.Time
	IsStart bool
	Project string
	Note    string
}

func buildCardSQL(isPunchIn bool, client string, note string) *CardSchemaSQL {
	if len(client) < 1 {
		panic("tried to build CardSchemaSQL object without required Project field")
	}

	punchAsInt := 0
	if isPunchIn {
		punchAsInt = 1
	}
	return &CardSchemaSQL{
		Punch:   int(time.Now().Unix()),
		Status:  punchAsInt,
		Project: client,
		Note:    toNullString(note),
	}
}

// TODO(zacsh) replace with `CardSchema{}` literal expression
var emptyCard CardSchema

func (c *CardSchema) isEmptyCard() bool {
	return *c == emptyCard
}

func (raw *CardSchemaSQL) toCard() *CardSchema {
	return &CardSchema{
		Punch:   time.Unix(int64(raw.Punch), 0 /*nanoseconds*/),
		IsStart: raw.Status == 1,
		Project: raw.Project,
		Note:    strings.TrimSpace(fromNullString(raw.Note)),
	}
}

func (card *CardSchema) toSQL() *CardSchemaSQL {
	var statusNum int
	if card.IsStart {
		statusNum = 1
	}

	return &CardSchemaSQL{
		Punch:   int(card.Punch.Unix()),
		Status:  statusNum,
		Project: card.Project,
		Note:    toNullString(strings.TrimSpace(card.Note)),
	}
}

type Session struct {
	StartAt   time.Time
	StopAt    time.Time
	Duration  time.Duration
	NoteStart string
	NoteStop  string
}

func (from *CardSchema) toSession(to *CardSchema) *Session {
	return &Session{
		StartAt:   from.Punch,
		StopAt:    to.Punch,
		Duration:  to.Punch.Sub(from.Punch),
		NoteStart: from.Note,
		NoteStop:  to.Note,
	}
}

func (s *Session) String() string {
	format := fmt.Sprintf("%s%d%s", "%", durationToStrMaxLen, "s from %s to %s%s")

	outPunchFormat := "15:04:05.9999"
	if s.Duration > time.Hour*22 {
		outPunchFormat = "01-02" + outPunchFormat
	}

	var notes string
	if len(s.NoteStart) > 0 {
		notes = fmt.Sprintf(" %s", s.NoteStart)
	}
	if len(s.NoteStop) > 0 {
		var separator string
		if len(s.NoteStart) > 0 {
			separator = ";"
		}
		notes += fmt.Sprintf("%s %s", separator, s.NoteStop)
	}

	return fmt.Sprintf(format,
		s.durationToStr(),
		s.StartAt.Format(format_dateTime),
		s.StopAt.Format(outPunchFormat),
		notes)
}
