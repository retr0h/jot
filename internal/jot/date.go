// Copyright (c) 2026 John Dewey
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package jot

import (
	"fmt"
	"strings"
	"time"
)

var dayNames = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"sun":       time.Sunday,
	"monday":    time.Monday,
	"mon":       time.Monday,
	"tuesday":   time.Tuesday,
	"tue":       time.Tuesday,
	"wednesday": time.Wednesday,
	"wed":       time.Wednesday,
	"thursday":  time.Thursday,
	"thu":       time.Thursday,
	"friday":    time.Friday,
	"fri":       time.Friday,
	"saturday":  time.Saturday,
	"sat":       time.Saturday,
}

// ParseDate parses a natural language or explicit date string relative to the
// given reference time. Supported inputs:
//
//   - "YYYY-MM-DD": parsed as an explicit date
//   - "today": the same date as relativeTo
//   - "tomorrow": relativeTo + 1 day
//   - day name (e.g. "friday"): the next occurrence of that weekday, always
//     in the future (if today is that day, the following week is returned)
//   - "next week": the coming Monday relative to relativeTo
func ParseDate(input string, relativeTo time.Time) (time.Time, error) {
	normalized := strings.ToLower(strings.TrimSpace(input))

	switch normalized {
	case "today":
		return truncateToDay(relativeTo), nil
	case "tomorrow":
		return truncateToDay(relativeTo).AddDate(0, 0, 1), nil
	case "next week":
		return nextWeekday(relativeTo, time.Monday), nil
	}

	// Explicit YYYY-MM-DD.
	if t, err := time.Parse("2006-01-02", normalized); err == nil {
		return t, nil
	}

	// Named weekday.
	if wd, ok := dayNames[normalized]; ok {
		return nextWeekday(relativeTo, wd), nil
	}

	// "next <day>" — the occurrence after the immediate next one.
	if strings.HasPrefix(normalized, "next ") {
		day := strings.TrimPrefix(normalized, "next ")
		if wd, ok := dayNames[day]; ok {
			first := nextWeekday(relativeTo, wd)
			return first.AddDate(0, 0, 7), nil
		}
	}

	return time.Time{}, fmt.Errorf("jot: unrecognized date input %q", input)
}

// truncateToDay returns a time.Time with only the date components of t,
// using the same location.
func truncateToDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// nextWeekday returns the next occurrence of wd strictly after relativeTo's
// weekday. If today is already wd, the result is one week from today.
func nextWeekday(relativeTo time.Time, wd time.Weekday) time.Time {
	base := truncateToDay(relativeTo)
	days := int(wd-base.Weekday()+7) % 7
	if days == 0 {
		days = 7
	}
	return base.AddDate(0, 0, days)
}
