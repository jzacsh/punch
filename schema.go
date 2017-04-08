package main

import (
	"fmt"
	"strings"
	"time"
)

type CardSchemaRaw struct {
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

func (raw *CardSchemaRaw) toCard() *CardSchema {
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

//////////////////////////////////
// Pretty printers for above datas

func (s *Session) durationToStr() string {
	return durationToStr(s.Duration)
}

func durationToStr(d time.Duration) string {
	daysStr := ""
	days := int(d.Hours()) / 24
	if days > 0 {
		daysStr = fmt.Sprintf("%f days ", days)
	}
	h, m, s := durationToHMS(d)
	colonIf := func(q int) string {
		if q > 0 {
			return fmt.Sprintf("%02d:", q)
		}
		return ""
	}
	return fmt.Sprintf("%s%s%02d:%02d", daysStr, colonIf(h), m, s)
}

func durationToHMS(d time.Duration) (int, int, int) {
	days := int(d.Hours()) / 24
	h := int((d - time.Duration(days)*time.Hour*24).Hours()) % 24
	m := int((d - time.Duration(days)*time.Hour*24 -
		time.Duration(h)*time.Hour).Minutes())
	s := int((d - time.Duration(days)*time.Hour*24 -
		time.Duration(h)*time.Hour -
		time.Duration(m)*time.Minute).Seconds())
	return h, m, s
}

func fromStatus(status bool) string {
	if status {
		return "in"
	}
	return "out"
}

func fromNote(note string) string {
	if len(note) == 0 {
		return "n/a"
	}
	return note
}
