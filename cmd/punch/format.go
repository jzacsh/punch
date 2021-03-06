package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const format_dateTime string = "2006-01-02 15:04:05"

func (s *Session) durationToStr() string {
	return durationToStr(s.Duration)
}

// TODO(zacsh) dry up places where this is done by hand
func parseStampCommand(cmd string) (time.Time, error) {
	stamp, e := strconv.ParseInt(strings.TrimSpace(cmd), 10, 64)
	if e != nil {
		return time.Time{}, fmt.Errorf("stamp arg: %s", e)
	}
	return time.Unix(stamp, 0 /*nanoseconds*/), nil
}

const durationToStrMaxLen = 20

func getTZContext() string {
	return time.Now().Format("-0700 MST")
}

func durationToStr(d time.Duration) string {
	daysStr := ""
	days := int(d.Hours()) / 24
	if days > 0 {
		daysStr = fmt.Sprintf("%04d days ", days)
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

func isValidClient(clientStr string) bool {
	if len(clientStr) < 1 {
		return false
	}

	alphaOrNumeric := "[[:alpha:]]|[[:digit:]]"
	validRegexp := regexp.MustCompile(fmt.Sprintf(
		"^(%s)+(-*%s)*(_*%s)*$", alphaOrNumeric, alphaOrNumeric, alphaOrNumeric))
	return validRegexp.MatchString(clientStr)
}
