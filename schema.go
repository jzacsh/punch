package main

import (
	"strings"
	"time"
)

type CardSchemaSQL struct {
	Punch   int // unix stamp seconds; primary key
	Status  int // (pseudo-boolean) 1,0
	Project string
	Note    string
}

type CardSchema struct {
	Punch   time.Time
	IsStart bool
	Project string
	Note    string
}

func buildCardSQL(isPunchIn bool, client string, note string) *CardSchemaSQL {
	punchAsInt := 0
	if isPunchIn {
		punchAsInt = 1
	}
	return &CardSchemaSQL{
		Punch:   int(time.Now().Unix()),
		Status:  punchAsInt,
		Project: client,
		Note:    note,
	}
}

var emptyCard CardSchema

func (c *CardSchema) isEmptyCard() bool {
	return *c == emptyCard
}

func (raw *CardSchemaSQL) toCard() *CardSchema {
	return &CardSchema{
		Punch:   time.Unix(int64(raw.Punch), 0 /*nanoseconds*/),
		IsStart: raw.Status == 1,
		Project: raw.Project,
		Note:    strings.TrimSpace(raw.Note),
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

///////////////////////////////////////////////////////////
// Not *our* schema, but schema manipulation nonetheless...

func isEmptyTime(t *time.Time) bool {
	var defaultTime time.Time
	return t.Sub(defaultTime) == 0
}
