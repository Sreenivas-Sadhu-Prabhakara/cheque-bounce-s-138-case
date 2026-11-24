package s138

import (
	"encoding/json"
	"testing"
	"time"
)

// mustDay parses a YYYY-MM-DD date in a test, or fails the test.
func mustDay(t *testing.T, s string) Day {
	t.Helper()
	d, err := ParseDay(s)
	if err != nil {
		t.Fatalf("ParseDay(%q): %v", s, err)
	}
	return d
}

func TestAddDaysAcrossMonthEndsAndLeapDays(t *testing.T) {
	cases := []struct {
		name string
		from string
		days int
		want string
	}{
		// The thirty days of s.138 proviso (b) off a 31-day month, in a leap
		// year and in an ordinary one. One day apart, and both have to be right.
		{"memo 31 Jan, leap year, +30", "2024-01-31", 30, "2024-03-01"},
		{"memo 31 Jan, ordinary year, +30", "2023-01-31", 30, "2023-03-02"},
		{"memo 1 Feb, leap year, +30", "2024-02-01", 30, "2024-03-02"},
		{"memo 1 Feb, ordinary year, +30", "2023-02-01", 30, "2023-03-03"},

		// The fifteen days of proviso (c) landing on and around 29 February.
		{"served 14 Feb, leap year, +15", "2024-02-14", 15, "2024-02-29"},
		{"served 14 Feb, ordinary year, +15", "2023-02-14", 15, "2023-03-01"},
		{"served 15 Feb, leap year, +15", "2024-02-15", 15, "2024-03-01"},
		{"served 16 Jan, +15 lands on 31 Jan", "2024-01-16", 15, "2024-01-31"},
		{"served 16 Jan, +16 lands on 1 Feb", "2024-01-16", 16, "2024-02-01"},

		// Leap-day arithmetic itself.
		{"28 Feb to 29 Feb in a leap year", "2024-02-28", 1, "2024-02-29"},
		{"28 Feb to 1 Mar in an ordinary year", "2023-02-28", 1, "2023-03-01"},
		{"29 Feb to 1 Mar", "2024-02-29", 1, "2024-03-01"},
		{"1900 was not a leap year", "1900-02-28", 1, "1900-03-01"},
		{"2000 was a leap year", "2000-02-28", 1, "2000-02-29"},

		// Year ends and zero and negative days.
		{"across the new year", "2024-12-31", 1, "2025-01-01"},
		{"31 Dec + 30", "2024-12-15", 30, "2025-01-14"},
		{"zero days is the same day", "2024-02-29", 0, "2024-02-29"},
		{"negative days reverse exactly", "2024-03-01", -30, "2024-01-31"},
		{"negative across the leap day", "2024-03-01", -1, "2024-02-29"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mustDay(t, c.from).AddDays(c.days)
			if got.String() != c.want {
				t.Errorf("%s + %d days = %s, want %s", c.from, c.days, got, c.want)
			}
		})
	}
}

func TestAddMonthsClampsToTheEndOfTheMonth(t *testing.T) {
	cases := []struct {
		name   string
		from   string
		months int
		want   string
	}{
		// The whole reason this method exists: time.AddDate would normalise
		// these forward into the following month and hand the user days of
		// limitation that do not exist.
		{"31 Jan + 1 month in a leap year", "2024-01-31", 1, "2024-02-29"},
		{"31 Jan + 1 month in an ordinary year", "2023-01-31", 1, "2023-02-28"},
		{"30 Jan + 1 month in a leap year", "2024-01-30", 1, "2024-02-29"},
		{"29 Jan + 1 month in a leap year", "2024-01-29", 1, "2024-02-29"},
		{"28 Jan + 1 month needs no clamp", "2024-01-28", 1, "2024-02-28"},
		{"29 Jan + 1 month in an ordinary year", "2023-01-29", 1, "2023-02-28"},

		// Every 31-day month rolling into a 30-day one.
		{"31 Mar + 1 month", "2024-03-31", 1, "2024-04-30"},
		{"31 May + 1 month", "2024-05-31", 1, "2024-06-30"},
		{"31 Aug + 1 month", "2024-08-31", 1, "2024-09-30"},
		{"31 Oct + 1 month", "2024-10-31", 1, "2024-11-30"},
		{"31 Dec + 1 month needs no clamp", "2024-12-31", 1, "2025-01-31"},

		// Longer runs, used for cheque validity.
		{"30 Nov + 3 months into a short February", "2024-11-30", 3, "2025-02-28"},
		{"30 Nov + 3 months into a leap February", "2023-11-30", 3, "2024-02-29"},
		{"31 Jan + 3 months", "2024-01-31", 3, "2024-04-30"},
		{"31 Jan + 6 months", "2024-01-31", 6, "2024-07-31"},
		{"31 Dec + 3 months", "2023-12-31", 3, "2024-03-31"},
		{"29 Feb + 1 month", "2024-02-29", 1, "2024-03-29"},
		{"15 Jan + 1 month needs no clamp", "2024-01-15", 1, "2024-02-15"},

		// Across a year boundary, backwards, and zero.
		{"31 Dec 1999 + 1 month", "1999-12-31", 1, "2000-01-31"},
		{"31 Mar - 1 month clamps too", "2024-03-31", -1, "2024-02-29"},
		{"31 Mar - 1 month, ordinary year", "2023-03-31", -1, "2023-02-28"},
		{"zero months is the same day", "2024-01-31", 0, "2024-01-31"},
		{"12 months is a year", "2024-01-31", 12, "2025-01-31"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mustDay(t, c.from).AddMonths(c.months)
			if got.String() != c.want {
				t.Errorf("%s + %d months = %s, want %s", c.from, c.months, got, c.want)
			}
		})
	}
}

// TestAddMonthsIsNotThirtyDays pins the distinction the whole tool turns on: a
// calendar month is not thirty days, and in a short month it is shorter.
func TestAddMonthsIsNotThirtyDays(t *testing.T) {
	cases := []struct {
		from       string
		wantMonth  string
		want30Days string
		shorter    bool // is the calendar month the EARLIER of the two?
	}{
		{"2023-02-01", "2023-03-01", "2023-03-03", true},
		{"2024-02-01", "2024-03-01", "2024-03-02", true},
		{"2024-04-01", "2024-05-01", "2024-05-01", false},
		{"2024-01-01", "2024-02-01", "2024-01-31", false},
		{"2024-03-01", "2024-04-01", "2024-03-31", false},
	}
	for _, c := range cases {
		t.Run(c.from, func(t *testing.T) {
			from := mustDay(t, c.from)
			month, thirty := from.AddMonths(1), from.AddDays(30)
			if month.String() != c.wantMonth {
				t.Errorf("one month from %s = %s, want %s", c.from, month, c.wantMonth)
			}
			if thirty.String() != c.want30Days {
				t.Errorf("thirty days from %s = %s, want %s", c.from, thirty, c.want30Days)
			}
			if got := month.Before(thirty); got != c.shorter {
				t.Errorf("one month from %s earlier than thirty days = %v, want %v", c.from, got, c.shorter)
			}
		})
	}
}

func TestAddYearsClampsTheLeapDay(t *testing.T) {
	cases := []struct {
		from  string
		years int
		want  string
	}{
		{"2024-02-29", 1, "2025-02-28"},
		{"2024-02-29", 2, "2026-02-28"},
		{"2024-02-29", 3, "2027-02-28"},
		{"2024-02-29", 4, "2028-02-29"},
		{"2000-02-29", 1, "2001-02-28"},
		{"2023-11-23", 3, "2026-11-23"},
		{"2023-12-31", 3, "2026-12-31"},
		{"1900-02-28", 100, "2000-02-28"},
		{"2024-02-28", 1, "2025-02-28"},
	}
	for _, c := range cases {
		t.Run(c.from, func(t *testing.T) {
			got := mustDay(t, c.from).AddYears(c.years)
			if got.String() != c.want {
				t.Errorf("%s + %d years = %s, want %s", c.from, c.years, got, c.want)
			}
		})
	}
}

func TestDaysInMonth(t *testing.T) {
	cases := []struct {
		year  int
		month time.Month
		want  int
	}{
		{2024, time.February, 29},
		{2023, time.February, 28},
		{2000, time.February, 29},
		{1900, time.February, 28},
		{2100, time.February, 28},
		{2024, time.January, 31},
		{2024, time.April, 30},
		{2024, time.December, 31},
	}
	for _, c := range cases {
		if got := DaysInMonth(c.year, c.month); got != c.want {
			t.Errorf("DaysInMonth(%d, %s) = %d, want %d", c.year, c.month, got, c.want)
		}
	}
}

func TestDaysUntil(t *testing.T) {
	cases := []struct {
		from, to string
		want     int
	}{
		{"2024-02-28", "2024-03-01", 2}, // the leap day sits in between
		{"2023-02-28", "2023-03-01", 1},
		{"2024-01-31", "2024-03-01", 30},
		{"2023-01-31", "2023-03-01", 29},
		{"2024-02-29", "2024-02-29", 0},
		{"2024-03-01", "2024-02-28", -2},
		{"2024-01-01", "2025-01-01", 366},
		{"2023-01-01", "2024-01-01", 365},
	}
	for _, c := range cases {
		if got := mustDay(t, c.from).DaysUntil(mustDay(t, c.to)); got != c.want {
			t.Errorf("%s until %s = %d, want %d", c.from, c.to, got, c.want)
		}
	}
	if got := (Day{}).DaysUntil(mustDay(t, "2024-01-01")); got != 0 {
		t.Errorf("an unrecorded date counts zero days, got %d", got)
	}
}

func TestParseDayRejectsImpossibleDates(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
		want    string
	}{
		{"2024-02-29", false, "2024-02-29"},
		{"", false, ""},
		{"   ", false, ""},
		{" 2024-01-31 ", false, "2024-01-31"},
		{"2023-02-29", true, ""}, // not a leap year
		{"2024-02-30", true, ""},
		{"2024-04-31", true, ""},
		{"2024-13-01", true, ""},
		{"2024-00-10", true, ""},
		{"2024-1-9", true, ""},
		{"31-01-2024", true, ""},
		{"tomorrow", true, ""},
	}
	for _, c := range cases {
		got, err := ParseDay(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseDay(%q) = %s, want an error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDay(%q): unexpected error %v", c.in, err)
			continue
		}
		if got.String() != c.want {
			t.Errorf("ParseDay(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDayJSONRoundTrip(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{`"2024-02-29"`, "2024-02-29"},
		{`""`, ""},
		{`null`, ""},
	}
	for _, c := range cases {
		var d Day
		if err := json.Unmarshal([]byte(c.raw), &d); err != nil {
			t.Fatalf("unmarshal %s: %v", c.raw, err)
		}
		if d.String() != c.want {
			t.Errorf("unmarshal %s = %q, want %q", c.raw, d, c.want)
		}
		out, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("marshal %q: %v", d, err)
		}
		if string(out) != `"`+c.want+`"` {
			t.Errorf("marshal %q = %s, want %q", d, out, `"`+c.want+`"`)
		}
	}
	var d Day
	if err := json.Unmarshal([]byte(`"2024-02-30"`), &d); err == nil {
		t.Error("unmarshalling an impossible date should fail")
	}
	if err := json.Unmarshal([]byte(`17`), &d); err == nil {
		t.Error("unmarshalling a number as a date should fail")
	}
}

func TestPrettyAndCompact(t *testing.T) {
	d := mustDay(t, "2024-02-29")
	if got, want := d.Pretty(), "29 February 2024"; got != want {
		t.Errorf("Pretty() = %q, want %q", got, want)
	}
	if got, want := d.Compact(), "20240229"; got != want {
		t.Errorf("Compact() = %q, want %q", got, want)
	}
	if got, want := (Day{}).Pretty(), "—"; got != want {
		t.Errorf("an unrecorded date prints as %q, want %q", got, want)
	}
	if got := (Day{}).Compact(); got != "" {
		t.Errorf("an unrecorded date has no compact form, got %q", got)
	}
}

func TestIsWeekend(t *testing.T) {
	cases := []struct {
		day  string
		want bool
	}{
		{"2024-01-05", false}, // Friday
		{"2024-01-06", true},  // Saturday
		{"2024-01-07", true},  // Sunday
		{"2024-01-08", false}, // Monday
		{"2024-02-29", false}, // Thursday
	}
	for _, c := range cases {
		d := mustDay(t, c.day)
		if got := d.IsWeekend(); got != c.want {
			t.Errorf("%s is a %s; IsWeekend() = %v, want %v", c.day, d.Weekday(), got, c.want)
		}
	}
}

func TestEarlierAndLaterIgnoreUnrecordedDates(t *testing.T) {
	a := mustDay(t, "2024-01-31")
	b := mustDay(t, "2024-03-01")
	zero := Day{}

	if got := Earlier(a, b); !got.Equal(a) {
		t.Errorf("Earlier(%s, %s) = %s, want %s", a, b, got, a)
	}
	if got := Earlier(b, a); !got.Equal(a) {
		t.Errorf("Earlier(%s, %s) = %s, want %s", b, a, got, a)
	}
	if got := Earlier(zero, b); !got.Equal(b) {
		t.Errorf("Earlier(unrecorded, %s) = %s, want %s", b, got, b)
	}
	if got := Earlier(a, zero); !got.Equal(a) {
		t.Errorf("Earlier(%s, unrecorded) = %s, want %s", a, got, a)
	}
	if got := Later(a, b); !got.Equal(b) {
		t.Errorf("Later(%s, %s) = %s, want %s", a, b, got, b)
	}
	if got := Later(zero, a); !got.Equal(a) {
		t.Errorf("Later(unrecorded, %s) = %s, want %s", a, got, a)
	}
	if got := Earlier(zero, zero); !got.IsZero() {
		t.Errorf("Earlier of two unrecorded dates should be unrecorded, got %s", got)
	}
}

func TestZeroDayArithmeticStaysZero(t *testing.T) {
	zero := Day{}
	for _, got := range []Day{zero.AddDays(30), zero.AddMonths(1), zero.AddYears(3)} {
		if !got.IsZero() {
			t.Errorf("arithmetic on an unrecorded date produced %s; it must stay unrecorded", got)
		}
	}
}

func TestDayOfTruncatesTheClock(t *testing.T) {
	ts := time.Date(2024, time.February, 29, 23, 59, 59, 0, time.FixedZone("IST", 19800))
	if got, want := DayOf(ts).String(), "2024-02-29"; got != want {
		t.Errorf("DayOf(%s) = %s, want %s", ts, got, want)
	}
}

func TestOnOrBefore(t *testing.T) {
	a := mustDay(t, "2024-02-29")
	cases := []struct {
		other string
		want  bool
	}{
		{"2024-02-28", false},
		{"2024-02-29", true},
		{"2024-03-01", true},
	}
	for _, c := range cases {
		if got := a.OnOrBefore(mustDay(t, c.other)); got != c.want {
			t.Errorf("%s.OnOrBefore(%s) = %v, want %v", a, c.other, got, c.want)
		}
	}
}
