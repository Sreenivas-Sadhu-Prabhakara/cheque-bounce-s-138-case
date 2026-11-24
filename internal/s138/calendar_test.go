package s138

import (
	"strings"
	"testing"
)

// unfold reverses RFC 5545 line folding, so a test can read a content line
// whole the way a calendar application does.
func unfold(ics string) string {
	return strings.ReplaceAll(ics, "\r\n ", "")
}

func icsLines(t *testing.T, ics string) []string {
	t.Helper()
	if !strings.HasSuffix(ics, "\r\n") {
		t.Fatal("the calendar must end with CRLF")
	}
	body := strings.TrimSuffix(unfold(ics), "\r\n")
	lines := strings.Split(body, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, "\n") {
			t.Fatalf("a content line contains a bare newline: %q", line)
		}
	}
	return lines
}

func TestCalendarICSStructure(t *testing.T) {
	c := testCase(t)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	ics := CalendarICS(c, cas, mustDay(t, "2024-02-01"))

	if !strings.HasPrefix(ics, "BEGIN:VCALENDAR\r\n") {
		t.Error("the calendar must open with BEGIN:VCALENDAR")
	}
	if !strings.HasSuffix(ics, "END:VCALENDAR\r\n") {
		t.Error("the calendar must close with END:VCALENDAR")
	}
	lines := icsLines(t, ics)
	for _, want := range []string{"VERSION:2.0", "CALSCALE:GREGORIAN", "METHOD:PUBLISH"} {
		if !contains(lines, want) {
			t.Errorf("the calendar is missing %q", want)
		}
	}
	if begins, ends := countLine(lines, "BEGIN:VEVENT"), countLine(lines, "END:VEVENT"); begins != ends {
		t.Errorf("%d events opened and %d closed", begins, ends)
	}
	if begins, ends := countLine(lines, "BEGIN:VALARM"), countLine(lines, "END:VALARM"); begins != ends {
		t.Errorf("%d alarms opened and %d closed", begins, ends)
	}
}

func contains(lines []string, want string) bool {
	for _, l := range lines {
		if l == want {
			return true
		}
	}
	return false
}

func countLine(lines []string, want string) int {
	n := 0
	for _, l := range lines {
		if l == want {
			n++
		}
	}
	return n
}

// TestCalendarICSFoldsEveryLineTo75Octets is what makes the file importable:
// an unfolded 300-character DESCRIPTION is rejected or truncated by some
// calendar applications.
func TestCalendarICSFoldsEveryLineTo75Octets(t *testing.T) {
	c := testCase(t)
	// A deliberately long reference, to force folding.
	c.Ref = strings.Repeat("NI/2024/0007-LONG-FILE-REFERENCE-", 4)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	ics := CalendarICS(c, cas, mustDay(t, "2024-02-01"))

	raw := strings.Split(strings.TrimSuffix(ics, "\r\n"), "\r\n")
	for _, line := range raw {
		if len(line) > 75 {
			t.Errorf("a folded line is %d octets long: %q", len(line), line)
		}
	}
	// Folding must be lossless: the long reference comes back whole.
	if !strings.Contains(unfold(ics), c.Ref) {
		t.Error("unfolding the calendar should give the reference back whole")
	}
}

func TestCalendarICSEventsAndDates(t *testing.T) {
	c := testCase(t)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	ics := CalendarICS(c, cas, mustDay(t, "2024-02-01"))
	lines := icsLines(t, ics)

	// On 1 February 2024 the cheque was presented and the notice served, so
	// the only deadlines still ahead are the complaint and the civil suit.
	if got, want := countLine(lines, "BEGIN:VEVENT"), 2; got != want {
		t.Errorf("%d events, want %d:\n%s", got, want, unfold(ics))
	}
	for _, want := range []string{
		"DTSTART;VALUE=DATE:20240229", // the complaint deadline
		"DTEND;VALUE=DATE:20240301",   // all-day DTEND is exclusive
		"DTSTART;VALUE=DATE:20261222", // the civil track
		"DTSTAMP:20240201T000000Z",
	} {
		if !contains(lines, want) {
			t.Errorf("the calendar is missing %q", want)
		}
	}
	// Four reminders on each of the two events.
	if got, want := countLine(lines, "ACTION:DISPLAY"), 2*len(alarmLeads); got != want {
		t.Errorf("%d reminders, want %d", got, want)
	}
	for _, lead := range alarmLeads {
		if !contains(lines, "TRIGGER:"+lead) {
			t.Errorf("the calendar has no reminder at %s", lead)
		}
	}

	text := unfold(ics)
	for _, want := range []string{
		"File the s.138 complaint",
		"Veda Fabricators Private Limited",
		"s.142(1)(b)",
		"Missing this window can extinguish the complaint",
		"Confirm the period and your facts with an advocate",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the calendar does not mention %q", want)
		}
	}
}

func TestCalendarICSSkipsWhatIsDoneMissedOrNotYours(t *testing.T) {
	c := testCase(t)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	text := unfold(CalendarICS(c, cas, mustDay(t, "2024-02-01")))

	// The presentation and the notice are done; the drawer's fifteen days are
	// the drawer's problem, not a diary entry for the payee.
	for _, unwanted := range []string{
		"Present the cheque",
		"Serve the s.138 demand notice",
		"Drawer's fifteen days",
	} {
		if strings.Contains(text, unwanted) {
			t.Errorf("the calendar should not carry %q once it is done or not the payee's step", unwanted)
		}
	}

	// A case where every window has closed produces no events at all: there is
	// nothing left to diarise.
	dead := testCase(t)
	dead.NoticeSentDate, dead.NoticeServedDate = Day{}, Day{}
	deadCas := dead.Cascade(DefaultPeriods(), mustDay(t, "2024-06-01"))
	deadICS := CalendarICS(dead, deadCas, mustDay(t, "2024-06-01"))
	if strings.Contains(deadICS, "Serve the s.138 demand notice") {
		t.Error("a window that has closed is not a reminder")
	}

	// A cheque paid inside the fifteen days leaves nothing to diarise at all.
	paid := testCase(t)
	paid.PaidDate = mustDay(t, "2024-01-25")
	paidCas := paid.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	paidICS := CalendarICS(paid, paidCas, mustDay(t, "2024-02-01"))
	if strings.Contains(paidICS, "BEGIN:VEVENT") {
		t.Errorf("a paid cheque needs no reminders:\n%s", unfold(paidICS))
	}
}

func TestCalendarICSBeforeTheNoticeGoesOut(t *testing.T) {
	c := testCase(t)
	c.NoticeSentDate, c.NoticeServedDate = Day{}, Day{}
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2023-12-26"))
	ics := CalendarICS(c, cas, mustDay(t, "2023-12-26"))
	lines := icsLines(t, ics)

	// The notice deadline and the presentation deadline are both still ahead;
	// the complaint window cannot be computed yet, so it is not in the file.
	if !contains(lines, "DTSTART;VALUE=DATE:20240121") {
		t.Error("the notice deadline of 21 January 2024 should be in the calendar")
	}
	if strings.Contains(unfold(ics), "File the s.138 complaint") {
		t.Error("an uncomputable complaint deadline must not be put in a diary")
	}
}

func TestCalendarICSIsDeterministic(t *testing.T) {
	c := testCase(t)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	first := CalendarICS(c, cas, mustDay(t, "2024-02-01"))
	second := CalendarICS(c, cas, mustDay(t, "2024-02-01"))
	if first != second {
		t.Error("the same case on the same date must produce the same calendar")
	}
	if !strings.Contains(first, "UID:ni-2024-07-complaint-20240229@") {
		t.Errorf("the event UID should be stable and derived from the case:\n%s", unfold(first))
	}
}

func TestEscapeICS(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain", "plain"},
		{"a,b", `a\,b`},
		{"a;b", `a\;b`},
		{`a\b`, `a\\b`},
		{"line1\nline2", `line1\nline2`},
		{"line1\r\nline2", `line1\nline2`},
		{"line1\rline2", `line1\nline2`},
		{"Rs. 2,50,000.00; unpaid", `Rs. 2\,50\,000.00\; unpaid`},
	}
	for _, c := range cases {
		if got := escapeICS(c.in); got != c.want {
			t.Errorf("escapeICS(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFold(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"short line", "VERSION:2.0"},
		{"exactly 75 octets", strings.Repeat("a", 75)},
		{"76 octets", strings.Repeat("a", 76)},
		{"very long", "DESCRIPTION:" + strings.Repeat("the complaint must be filed within one calendar month. ", 12)},
		{"multi-byte runes", "SUMMARY:" + strings.Repeat("Rs. 2,50,000 — unpaid ", 10)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := fold(c.in)
			if !strings.HasSuffix(got, "\r\n") {
				t.Fatalf("fold(%q) does not end with CRLF", c.name)
			}
			for _, line := range strings.Split(strings.TrimSuffix(got, "\r\n"), "\r\n") {
				if len(line) > 75 {
					t.Errorf("fold left a %d-octet line: %q", len(line), line)
				}
			}
			// Folding then unfolding must return exactly what went in.
			if back := strings.TrimSuffix(unfold(got), "\r\n"); back != c.in {
				t.Errorf("fold is not lossless:\n got %q\nwant %q", back, c.in)
			}
		})
	}
}

func TestSlug(t *testing.T) {
	cases := []struct{ in, want string }{
		{"NI/2024/07", "ni-2024-07"},
		{"NI 2024 07", "ni-2024-07"},
		{"  cheque // 12  ", "cheque-12"},
		{"ABC123", "abc123"},
		{"", "case"},
		{"///", "case"},
		{"—", "case"},
	}
	for _, c := range cases {
		if got := slug(c.in); got != c.want {
			t.Errorf("slug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
