// Package s138 implements the statutory deadline cascade that follows the
// dishonour of a cheque under section 138 of the Negotiable Instruments Act,
// 1881, and the limitation period for the complaint under section 142.
//
// Every statutory period lives in one place, as a named constant carrying the
// section it comes from (see periods.go). Every deadline is derived from those
// constants by calendar-correct arithmetic in this package, as pure functions
// over plain values, so the rules can be read and tested without a server.
//
// The date arithmetic is the product. Missing any one of these windows
// extinguishes the complaint, so a wrong date here is the worst bug this code
// can have — worse than a crash, because a crash is visible. Two consequences
// for anyone changing this package:
//
//   - Months and years are added by clamping to the end of the target month,
//     never by letting the calendar roll over. One month from 31 January is 28
//     or 29 February, never 2 or 3 March. Go's time.AddDate normalises instead
//     of clamping, which would silently hand the user two extra days of
//     limitation that do not exist.
//   - Where the law or the date of an event is genuinely uncertain, the
//     cascade reports the EARLIER of the defensible dates as the deadline to
//     work to, and the later one beside it as the outer limit. Erring early
//     costs a few days; erring late costs the case.
package s138

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DayLayout is the calendar-date format used everywhere in this tool: in the
// JSON API, in the data file and in the ISO dates printed on documents.
const DayLayout = "2006-01-02"

// Day is a calendar date with no clock time and no time zone.
//
// Statutory periods under the Negotiable Instruments Act are counted in whole
// days and whole calendar months, so the domain deliberately has no notion of
// hours: every Day is held as midnight UTC. That makes day subtraction exact
// and immune to daylight-saving shifts, which would otherwise be able to move
// a deadline by a day.
//
// The zero Day means "not recorded" — a notice that has not been sent, a
// complaint that has not been filed — and encodes to an empty JSON string.
type Day struct {
	t time.Time
}

// NewDay builds a Day from a year, month and day of month. Out-of-range values
// normalise the way time.Date does (31 April becomes 1 May).
func NewDay(year int, month time.Month, day int) Day {
	return Day{t: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// DayOf truncates any time to the calendar date it falls on in its own
// location, so "today" is the user's today and not UTC's.
func DayOf(t time.Time) Day {
	y, m, d := t.Date()
	return NewDay(y, m, d)
}

// ParseDay reads a YYYY-MM-DD date. An empty or blank string yields the zero
// Day, which the domain reads as "not recorded".
func ParseDay(s string) (Day, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Day{}, nil
	}
	t, err := time.Parse(DayLayout, s)
	if err != nil {
		return Day{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", s)
	}
	return Day{t: t.UTC()}, nil
}

// IsZero reports whether the date is unrecorded.
func (d Day) IsZero() bool { return d.t.IsZero() }

// String renders the date as YYYY-MM-DD, or an empty string when unrecorded.
func (d Day) String() string {
	if d.IsZero() {
		return ""
	}
	return d.t.Format(DayLayout)
}

// Pretty renders the date the way it is written on a notice or a complaint,
// e.g. "02 April 2026". An unrecorded date prints as an em dash.
func (d Day) Pretty() string {
	if d.IsZero() {
		return "—"
	}
	return d.t.Format("02 January 2006")
}

// Compact renders the date as YYYYMMDD, the form an iCalendar DATE value takes.
func (d Day) Compact() string {
	if d.IsZero() {
		return ""
	}
	return d.t.Format("20060102")
}

// Time returns the underlying midnight-UTC instant.
func (d Day) Time() time.Time { return d.t }

// Weekday returns the day of the week.
func (d Day) Weekday() time.Weekday { return d.t.Weekday() }

// IsWeekend reports whether the date falls on a Saturday or a Sunday.
//
// This is a convenience flag for the user's diary, not a rule. Courts are
// closed on Sundays and on some Saturdays, and section 4 of the Limitation
// Act, 1963 allows a filing on the day the court reopens when the last day of
// the prescribed period falls on a day the court is closed. Nothing in this
// package relies on that: a deadline that lands on a weekend is still shown as
// the deadline, because planning to arrive on the reopening day means planning
// to arrive late.
func (d Day) IsWeekend() bool {
	w := d.Weekday()
	return w == time.Saturday || w == time.Sunday
}

// AddDays returns the date n days later (n may be negative). Day counts under
// the proviso to section 138 are plain running days, including holidays.
func (d Day) AddDays(n int) Day {
	if d.IsZero() {
		return d
	}
	return Day{t: d.t.AddDate(0, 0, n)}
}

// AddMonths returns the date n calendar months later, clamped to the last day
// of the target month: 31 January plus one month is 28 February, or 29
// February in a leap year — never 2 or 3 March.
//
// Section 142(1)(b) gives "one month" to file the complaint, and section 3(35)
// of the General Clauses Act, 1897 defines a month as one reckoned by the
// British calendar. A calendar month is 28, 29, 30 or 31 days long, so it is
// not interchangeable with 30 days. Clamping matters because time.AddDate
// normalises an impossible date forward: 31 January + 1 month would come back
// as 2 or 3 March and overstate the limitation period by two or three days.
func (d Day) AddMonths(n int) Day {
	if d.IsZero() {
		return d
	}
	year, month, dom := d.t.Date()
	// Shift from the first of the month so the month arithmetic can never roll
	// over on its own, then clamp the day of month to the target month's last.
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, n, 0)
	ty, tm, _ := first.Date()
	if last := DaysInMonth(ty, tm); dom > last {
		dom = last
	}
	return NewDay(ty, tm, dom)
}

// AddYears returns the date n years later, clamped the same way: 29 February
// plus one year is 28 February, not 1 March.
func (d Day) AddYears(n int) Day { return d.AddMonths(12 * n) }

// DaysInMonth returns the number of days in the given month, leap years
// included. Day 0 of the following month is the last day of this one.
func DaysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// Before reports whether d falls before o.
func (d Day) Before(o Day) bool { return d.t.Before(o.t) }

// After reports whether d falls after o.
func (d Day) After(o Day) bool { return d.t.After(o.t) }

// Equal reports whether d and o are the same calendar date.
func (d Day) Equal(o Day) bool { return d.t.Equal(o.t) }

// OnOrBefore reports whether d falls on or before o.
func (d Day) OnOrBefore(o Day) bool { return !d.t.After(o.t) }

// DaysUntil counts whole days from d to o. The result is negative when o falls
// before d, and zero when they are the same date.
func (d Day) DaysUntil(o Day) int {
	if d.IsZero() || o.IsZero() {
		return 0
	}
	return int(o.t.Sub(d.t) / (24 * time.Hour))
}

// Earlier returns whichever of the two dates comes first, ignoring an
// unrecorded one. The cascade uses it wherever the law offers two defensible
// readings and the safe choice is the earlier deadline.
func Earlier(a, b Day) Day {
	switch {
	case a.IsZero():
		return b
	case b.IsZero():
		return a
	case b.Before(a):
		return b
	default:
		return a
	}
}

// Later returns whichever of the two dates comes last, ignoring an unrecorded
// one.
func Later(a, b Day) Day {
	switch {
	case a.IsZero():
		return b
	case b.IsZero():
		return a
	case b.After(a):
		return b
	default:
		return a
	}
}

// MarshalJSON encodes the date as a YYYY-MM-DD string ("" when unrecorded).
func (d Day) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// UnmarshalJSON decodes a YYYY-MM-DD string; "" and null mean "not recorded".
func (d *Day) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*d = Day{}
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("date must be a YYYY-MM-DD string: %w", err)
	}
	parsed, err := ParseDay(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
