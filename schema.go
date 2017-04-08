package main

import "time"

type CardSchemaRaw struct {
	Punch   int // unix stamp seconds; primary key
	Status  int // (pseudo-boolean) 1,0
	Project string
	Note    string
}

type CardSchema struct {
	Punch   time.Time
	Status  bool
	Project string
	Note    string
}

func (raw *CardSchemaRaw) toCard() *CardSchema {
	return &CardSchema{
		Punch:   time.Unix(int64(raw.Punch), 0 /*nanoseconds*/),
		Status:  raw.Status == 1,
		Project: raw.Project,
		Note:    raw.Note,
	}
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
