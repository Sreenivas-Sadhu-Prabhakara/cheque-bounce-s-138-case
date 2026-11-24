package s138

import (
	"fmt"
	"strings"
)

// The reminders this tool produces are an iCalendar file you download and
// import into your own calendar. The tool has no way to send a message, an
// email or anything else, and no outbound network access at all — so the
// reminder has to live where you will actually see it, which is your own diary.

// alarmLeads are how far before a deadline each reminder fires, as iCalendar
// durations. A limitation deadline wants more than one warning.
var alarmLeads = []string{"-P14D", "-P7D", "-P3D", "-P1D"}

// icsProdID identifies the generator in the file, as RFC 5545 requires.
const icsProdID = "-//Open MSME//Cheque Bounce Section 138 Case Clock//EN"

// CalendarICS returns an iCalendar file holding one all-day event per live
// deadline in the case, each with reminders before it.
//
// Events are emitted only for deadlines that are still ahead or still
// actionable: there is no use putting a window that has closed into a diary.
// Line endings are CRLF and long lines are folded, as RFC 5545 requires, so
// the file imports into any calendar.
func CalendarICS(c Case, cas Cascade, asOf Day) string {
	var b strings.Builder
	write := func(s string) { b.WriteString(fold(s)) }

	write("BEGIN:VCALENDAR")
	write("VERSION:2.0")
	write("PRODID:" + icsProdID)
	write("CALSCALE:GREGORIAN")
	write("METHOD:PUBLISH")
	write("X-WR-CALNAME:" + escapeICS("Cheque bounce deadlines — "+Or(c.Ref, "case")))

	stamp := asOf.Compact() + "T000000Z"
	ref := Or(c.Ref, "case")
	seq := 0

	for _, s := range cas.Steps {
		if s.Due.IsZero() || s.Actor != "You" {
			continue
		}
		switch s.State {
		case StateDone, StateLate, StateMissed, StateUnknown, StateNA:
			continue
		}
		due := s.Due
		// The complaint event uses the date to work to, which can be earlier
		// than the step's own deadline while service is only assumed.
		if s.Key == "complaint" && !cas.FileBy.IsZero() {
			due = cas.FileBy
		}
		if due.Before(asOf) {
			continue
		}
		seq++
		uid := fmt.Sprintf("%s-%s-%s@open-msme.invalid", slug(ref), s.Key, due.Compact())
		summary := fmt.Sprintf("%s — %s (%s)", shortTitle(s.Key), Or(c.DrawerName, "cheque case"), ref)

		description := strings.Join([]string{
			s.Name + " — " + s.Section + ".",
			"Rule: " + s.Rule + ".",
			"Counted from " + s.From.Pretty() + " (" + s.FromWhat + ").",
			"Action: " + s.Action,
			outerLine(s),
			fmt.Sprintf("Cheque no. %s dated %s for Rs. %s, returned by memo dated %s.",
				Or(c.ChequeNo, "—"), c.ChequeDate.Pretty(), FormatINR(c.Amount), c.MemoDate.Pretty()),
			"Missing this window can extinguish the complaint. Confirm the period and your facts with an advocate.",
		}, "\n")

		write("BEGIN:VEVENT")
		write("UID:" + uid)
		write("DTSTAMP:" + stamp)
		write(fmt.Sprintf("SEQUENCE:%d", seq))
		write("DTSTART;VALUE=DATE:" + due.Compact())
		// DTEND on an all-day event is exclusive, so it is the following day.
		write("DTEND;VALUE=DATE:" + due.AddDays(1).Compact())
		write("SUMMARY:" + escapeICS(summary))
		write("DESCRIPTION:" + escapeICS(description))
		write("CATEGORIES:" + escapeICS("Statutory deadline"))
		write("TRANSP:TRANSPARENT")
		write("PRIORITY:1")
		for _, lead := range alarmLeads {
			write("BEGIN:VALARM")
			write("ACTION:DISPLAY")
			write("DESCRIPTION:" + escapeICS(summary))
			write("TRIGGER:" + lead)
			write("END:VALARM")
		}
		write("END:VEVENT")
	}

	write("END:VCALENDAR")
	return b.String()
}

func outerLine(s Step) string {
	if s.Outer.IsZero() {
		return "This is the date to work to."
	}
	return "The outer limit is " + s.Outer.Pretty() + "; this is the date to work to."
}

func shortTitle(key string) string {
	switch key {
	case "present":
		return "Present the cheque"
	case "notice":
		return "Serve the s.138 demand notice"
	case "complaint":
		return "File the s.138 complaint"
	case "civil":
		return "Civil suit limitation"
	default:
		return "Deadline"
	}
}

// escapeICS escapes the characters RFC 5545 reserves inside a text value.
func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	return s
}

// fold returns one content line terminated by CRLF, folded at 75 octets with a
// leading space on each continuation, as RFC 5545 requires. Folding counts
// octets, so it splits on byte boundaries that keep multi-byte runes whole.
func fold(line string) string {
	const limit = 75
	if len(line) <= limit {
		return line + "\r\n"
	}
	var b strings.Builder
	rest := line
	first := true
	for len(rest) > 0 {
		budget := limit
		if !first {
			budget = limit - 1 // the leading space counts
		}
		if len(rest) <= budget {
			if !first {
				b.WriteByte(' ')
			}
			b.WriteString(rest)
			b.WriteString("\r\n")
			break
		}
		cut := budget
		// Never cut in the middle of a UTF-8 rune.
		for cut > 0 && !isRuneStart(rest[cut]) {
			cut--
		}
		if cut == 0 {
			cut = budget
		}
		if !first {
			b.WriteByte(' ')
		}
		b.WriteString(rest[:cut])
		b.WriteString("\r\n")
		rest = rest[cut:]
		first = false
	}
	return b.String()
}

// isRuneStart reports whether b is the first byte of a UTF-8 sequence.
func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }

// slug reduces a file reference to something safe inside a UID: lower-case
// letters and digits, with runs of anything else collapsed to one hyphen.
func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
		default:
			dash = true
		}
	}
	if b.Len() == 0 {
		return "case"
	}
	return b.String()
}
