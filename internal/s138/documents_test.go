package s138

import (
	"strings"
	"testing"
)

func testOffice() Office {
	return Office{
		Advocate:  "A. N. Rao",
		Firm:      "Rao & Associates",
		Address:   "4th Floor, Sunrise Chambers, Bengaluru 560001",
		Phone:     "080 2222 3333",
		Email:     "office@example.invalid",
		Enrolment: "KAR/1234/2009",
	}
}

// testCase is the worked example used across the document tests: cheque for
// Rs 2,50,000 dated 1 December 2023, dishonoured by a memo of 22 December,
// notice out on 10 January 2024 and served on 16 January.
func testCase(t *testing.T) Case {
	t.Helper()
	return Case{
		ID:               "abc123",
		Ref:              "NI/2024/07",
		PayeeName:        "Kaveri Engineering Works",
		PayeeAddress:     "Plot 14, Peenya Industrial Area, Bengaluru 560058",
		DrawerName:       "Veda Fabricators Private Limited",
		DrawerAddress:    "22 Industrial Suburb, Yeshwanthpur, Bengaluru 560022",
		Consideration:    "goods supplied under invoices KEW/2023/118 and KEW/2023/126",
		ChequeNo:         "004512",
		Amount:           250000,
		DraweeBank:       "Shivamogga Co-operative Bank",
		DraweeBranch:     "Yeshwanthpur",
		CollectingBank:   "Union Bank of India",
		CollectingBranch: "Peenya",
		MemoReason:       "Funds insufficient",
		NoticeMode:       "registered post with A/D and speed post",
		Facts: Facts{
			ChequeDate:       mustDay(t, "2023-12-01"),
			PresentedDate:    mustDay(t, "2023-12-20"),
			MemoDate:         mustDay(t, "2023-12-22"),
			NoticeSentDate:   mustDay(t, "2024-01-10"),
			NoticeServedDate: mustDay(t, "2024-01-16"),
		},
	}
}

// flat collapses every run of whitespace to one space. The documents are
// wrapped to print on paper, so a phrase under test routinely straddles two
// lines; the tests care that it is in there, not where the fold fell.
func flat(s string) string { return strings.Join(strings.Fields(s), " ") }

// has reports whether the document contains the text, ignoring how it wrapped.
func has(document, want string) bool { return strings.Contains(flat(document), flat(want)) }

// fitsPage reports whether a printed line is within the page width. A line may
// run over only when it is a single unbreakable token — a file reference or a
// case citation — because cutting one of those in half is worse than a long
// line.
func fitsPage(line string) bool {
	return len([]rune(line)) <= docWidth || len(strings.Fields(line)) == 1
}

func longestLine(s string) int {
	longest := 0
	for _, line := range strings.Split(s, "\n") {
		if n := len([]rune(line)); n > longest {
			longest = n
		}
	}
	return longest
}

func TestDemandNoticeContainsWhatTheProvisoRequires(t *testing.T) {
	c := testCase(t)
	// Before the notice goes out, which is when this document is produced.
	c.NoticeSentDate, c.NoticeServedDate = Day{}, Day{}
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2023-12-26"))
	out := DemandNotice(testOffice(), c, cas, mustDay(t, "2023-12-26"))

	for _, want := range []string{
		// The demand itself, in the words clause (b) requires.
		"clause (b) of the proviso to section 138",
		"Negotiable Instruments Act, 1881",
		"WITHIN FIFTEEN (15) DAYS of the receipt of this notice",
		"pay to my client the said sum of Rs. 2,50,000.00",
		"Rupees Two Lakh Fifty Thousand only",
		// The particulars of the instrument and the dishonour.
		"cheque no. 004512 dated 01 December 2023",
		"Shivamogga Co-operative Bank, Yeshwanthpur branch",
		"Union Bank of India, Peenya branch",
		"memo dated 22 December 2023",
		"Funds insufficient",
		"20 December 2023",
		// The parties.
		"Veda Fabricators Private Limited",
		"Yeshwanthpur",
		"Kaveri Engineering Works",
		"goods supplied under invoices KEW/2023/118",
		// The consequence, from the named constants.
		"2 years",
		"twice the amount of the cheque",
		// How to send it, and the deadline to send it by.
		"BY REGISTERED POST WITH ACKNOWLEDGEMENT DUE",
		"Post it by 21 January 2024",
		// The office block and the honesty note.
		"A. N. Rao",
		"KAR/1234/2009",
		NotSentNotice,
		Disclaimer,
	} {
		if !has(out, want) {
			t.Errorf("the demand notice does not contain %q", want)
		}
	}

	if n := longestLine(out); n > docWidth {
		t.Errorf("the notice has a line %d characters long, want at most %d", n, docWidth)
	}
}

// TestEveryDocumentFitsThePageWidth is the printability guard. These documents
// go into a court file on paper, and a line that runs off the page takes a date
// with it. It found real defects: unwrapped rule lines in the schedule ran to
// 146 characters.
func TestEveryDocumentFitsThePageWidth(t *testing.T) {
	// Long values in every field that reaches a document, to push the folding.
	long := testCase(t)
	long.Ref = "NI/2024/07-A-VERY-LONG-FILE-REFERENCE-INDEED"
	long.PayeeName = "Kaveri Engineering and Fabrication Works Private Limited"
	long.PayeeAddress = "Plot 14 and 15, Fourth Phase, Peenya Industrial Area, Bengaluru 560058"
	long.DrawerName = "Veda Fabricators and General Engineering Private Limited"
	long.DrawerAddress = "22 Industrial Suburb Second Stage Yeshwanthpur Bengaluru Karnataka 560022"
	long.Consideration = "goods supplied under invoices KEW/2023/118, KEW/2023/126 and KEW/2023/131 together with the running account between the parties"
	long.DraweeBank = "The Shivamogga District Central Co-operative Bank Limited"
	long.DraweeBranch = "Yeshwanthpur Industrial Suburb Second Stage"
	long.CollectingBank = "Union Bank of India (erstwhile Corporation Bank)"
	long.MemoReason = "Funds insufficient / exceeds arrangement / payment stopped by drawer"
	long.CourtName = "Court of the XVII Additional Chief Metropolitan Magistrate, Bengaluru"
	long.Amount = 123456789.99

	office := Office{
		Advocate:  "A. N. Rao Venkataramanappa",
		Firm:      "Rao, Venkataramanappa & Associates, Advocates and Legal Consultants",
		Address:   "Fourth Floor, Sunrise Chambers, 22 Ulsoor Road, Bengaluru 560042",
		Phone:     "080 2222 3333 / 080 2222 4444",
		Email:     "chambers@rao-and-associates.example.invalid",
		Enrolment: "KAR/1234/2009",
	}

	subjects := map[string]Case{"a worked case": testCase(t), "long values everywhere": long, "nothing filled in": {
		Ref: "X", DrawerName: "D", ChequeNo: "1", Amount: 1, Facts: Facts{MemoDate: mustDay(t, "2024-01-31")}}}

	for name, c := range subjects {
		// Three positions in the cascade, so every branch of every document is
		// exercised: before the notice, mid-window, and past every deadline.
		for _, asOfRaw := range []string{"2023-12-26", "2024-02-01", "2024-09-01"} {
			asOf := mustDay(t, asOfRaw)
			cas := c.Cascade(DefaultPeriods(), asOf)
			docs := map[string]string{
				"demand notice":         DemandNotice(office, c, cas, asOf),
				"deadline schedule":     DeadlineSchedule(office, c, cas, asOf),
				"complaint particulars": ComplaintParticulars(office, c, cas, asOf),
				"case file checklist":   AnnexureChecklist(office, c, cas, asOf),
			}
			for doc, out := range docs {
				for i, l := range strings.Split(out, "\n") {
					if fitsPage(l) {
						continue
					}
					t.Errorf("%s / %s on %s: line %d is %d characters long:\n%s",
						name, doc, asOfRaw, i+1, len([]rune(l)), l)
				}
			}
		}
	}
}

// TestDemandNoticeDemandsTheChequeAmountAndNothingElse guards the trap that
// sinks otherwise sound cases: clause (b) requires a demand for "the said
// amount of money", and a notice that bundles interest or other invoices into
// the demand can be read as not making it.
func TestDemandNoticeDemandsTheChequeAmountAndNothingElse(t *testing.T) {
	c := testCase(t)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2023-12-26"))
	out := DemandNotice(testOffice(), c, cas, mustDay(t, "2023-12-26"))

	demand := "pay to my client the said sum of Rs. " + FormatINR(c.Amount)
	if !has(out, demand) {
		t.Fatalf("the operative demand %q is missing", demand)
	}
	if !has(out, "Check that the amount demanded is exactly the amount of the cheque") {
		t.Error("the notice should warn against demanding more than the cheque amount")
	}
	for _, forbidden := range []string{"with interest at", "plus interest", "and costs of this notice"} {
		if has(out, forbidden) {
			t.Errorf("the demand must not add %q to the cheque amount", forbidden)
		}
	}
}

func TestDemandNoticeMarksTheGapsItCannotFill(t *testing.T) {
	bare := Case{
		Ref:        "NI/2024/08",
		DrawerName: "Someone",
		ChequeNo:   "",
		Amount:     1000,
		Facts:      Facts{MemoDate: mustDay(t, "2024-01-05")},
	}
	cas := bare.Cascade(DefaultPeriods(), mustDay(t, "2024-01-08"))
	out := DemandNotice(Office{}, bare, cas, mustDay(t, "2024-01-08"))

	for _, want := range []string{
		"[ state here exactly what the cheque was given for",
		"[ cheque number ]",
		"[ the drawer's full address",
		"[ your client's full name ]",
		"[ the drawee bank ]",
		"[ the reason printed on the return memo, word for word ]",
		"[ your name ]",
	} {
		if !has(out, want) {
			t.Errorf("a blank field should print as %q so it is obvious what is missing", want)
		}
	}
	// A missing memo reason earns an extra warning, because some technical
	// return reasons fall outside section 138 altogether.
	if !has(out, "exceeds arrangement") {
		t.Error("with no memo reason recorded the notice should explain which reasons are within section 138")
	}
}

func TestDeadlineScheduleShowsTheWholeCascadeWithItsSources(t *testing.T) {
	c := testCase(t)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	out := DeadlineSchedule(testOffice(), c, cas, mustDay(t, "2024-02-01"))

	for _, want := range []string{
		"SECTION 138 DEADLINE SCHEDULE",
		"NI/2024/07",
		"Veda Fabricators Private Limited",
		// Each rung with its section.
		"s.138 proviso (a)", "s.138 proviso (b)", "s.138 proviso (c)", "s.142(1)(b)",
		"Limitation Act, 1963",
		// The computed dates.
		"01 March 2024",    // last day to present
		"21 January 2024",  // notice deadline
		"31 January 2024",  // the drawer's fifteen days expire
		"29 February 2024", // complaint deadline
		"22 December 2026", // the civil track
		// The one date that matters, and the arithmetic behind it.
		"THE ONE DATE TO REMEMBER",
		"File the complaint on or before  29 February 2024",
		"outer limit   01 March 2024",
		// The periods the schedule ran on.
		"THE PERIODS THIS SCHEDULE WAS COMPUTED ON",
		"1 calendar month",
		"NOT a statutory period",
		NotSentNotice,
		Disclaimer,
	} {
		if !has(out, want) {
			t.Errorf("the schedule does not contain %q", want)
		}
	}
	if has(out, "NOTE: one or more of these periods has been changed") {
		t.Error("a schedule computed on the statutory figures must not carry the override note")
	}
}

func TestDeadlineScheduleSaysWhenThePeriodsWereChanged(t *testing.T) {
	p := DefaultPeriods()
	p.NoticeDays = 15
	c := testCase(t)
	cas := c.Cascade(p, mustDay(t, "2024-02-01"))
	out := DeadlineSchedule(testOffice(), c, cas, mustDay(t, "2024-02-01"))

	if !has(out, "NOTE: one or more of these periods has been changed") {
		t.Error("a schedule computed on changed periods must say so")
	}
	if !has(out, "15 days  —  s.138 proviso (b)") {
		t.Error("the schedule must state the figure it actually used")
	}
}

func TestDeadlineScheduleReportsClaimEndingProblems(t *testing.T) {
	c := testCase(t)
	c.NoticeSentDate, c.NoticeServedDate = Day{}, Day{}
	// Long past the thirty days for the notice.
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-03-01"))
	out := DeadlineSchedule(testOffice(), c, cas, mustDay(t, "2024-03-01"))

	if !has(out, "CLAIM-ENDING PROBLEMS") {
		t.Fatal("a missed window must be reported under its own heading")
	}
	for _, want := range []string{
		"the thirty-day window under section 138 proviso (b) closed on 21 January 2024",
		"presenting it again",
		"the civil suit on the cheque is a separate track",
	} {
		if !has(out, want) {
			t.Errorf("the schedule does not explain %q", want)
		}
	}
}

func TestDeadlineScheduleFlagsAWeekendDeadline(t *testing.T) {
	c := testCase(t)
	c.MemoDate = mustDay(t, "2023-12-07") // notice deadline falls on Saturday 6 January
	c.NoticeSentDate, c.NoticeServedDate = Day{}, Day{}
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2023-12-10"))
	out := DeadlineSchedule(testOffice(), c, cas, mustDay(t, "2023-12-10"))

	if !has(out, "[falls on a Saturday]") {
		t.Error("a deadline on a weekend should be flagged on the schedule")
	}
}

func TestComplaintParticularsSpellOutTheLimitationArithmetic(t *testing.T) {
	c := testCase(t)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	out := ComplaintParticulars(testOffice(), c, cas, mustDay(t, "2024-02-01"))

	for _, want := range []string{
		"COMPLAINT PARTICULARS",
		"NI/2024/07",
		"Rao & Associates",
		// The limitation working, line by line.
		"Notice received (or deemed received)       16 January 2024",
		"expiring     31 January 2024",
		"Cause of action arose on                   01 February 2024",
		"DATE TO FILE BY                            29 February 2024",
		"Outer limit on the better view             01 March 2024",
		// The authority for the reckoning.
		"Econ Antri Ltd v. Rom Industries Ltd, (2014) 11 SCC 769",
		"Saketh India Ltd v. India Securities Ltd, (1999) 3 SCC 1",
		"a calendar month reckoned from the cause of action",
		// The other provisions an advocate needs on the sheet.
		"Section 142(1)(a)", "section 141", "Section 142(2)(a)", "Section 142(1)(c)",
		"Section 143A", "Section 148", "Section 147",
		"Section 145(1)", "Section 146", "Section 139",
		// Twenty per cent of Rs 2,50,000.
		"Rs. 50,000.00",
		"FILE BY 29 FEBRUARY 2024",
		Disclaimer,
	} {
		if !has(out, want) {
			t.Errorf("the complaint particulars do not contain %q", want)
		}
	}
	if has(out, "IN THE COURT OF") {
		t.Error("this document is a particulars sheet, not a draft complaint")
	}

	// The limitation working is laid out in columns rather than wrapped, so
	// these lines are checked exactly as they print.
	for _, want := range []string{
		"    Notice received (or deemed received)       16 January 2024",
		"    Cause of action arose on                   01 February 2024",
		"    DATE TO FILE BY                            29 February 2024",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the limitation working should print the line %q", want)
		}
	}
}

func TestComplaintParticularsSayWhenServiceIsOnlyAssumed(t *testing.T) {
	c := testCase(t)
	c.NoticeServedDate = Day{}
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-01-20"))
	out := ComplaintParticulars(testOffice(), c, cas, mustDay(t, "2024-01-20"))

	for _, want := range []string{
		"Service is not yet proved",
		"17 January 2024",
		"Section 27 of the General Clauses Act, 1897",
		"not a statutory period",
	} {
		if !has(out, want) {
			t.Errorf("the particulars should disclose the assumption; %q is missing", want)
		}
	}
}

func TestAnnexureChecklistListsTheEssentialsAndTheWayBack(t *testing.T) {
	c := testCase(t)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	out := AnnexureChecklist(testOffice(), c, cas, mustDay(t, "2024-02-01"))

	for _, want := range []string{
		"CASE FILE CHECKLIST",
		"ESSENTIAL — the original cheque no. 004512",
		"ESSENTIAL — the original bank return or dishonour memo dated 22 December 2023",
		"section 145(1)",
		"Do not open a returned envelope",
		"board resolution",
		// What is left when a window has gone.
		"IF A WINDOW HAS BEEN MISSED",
		"MSR Leathers v. S. Palaniappan, (2013) 1 SCC 177",
		"on or before 01 March 2024", // the cheque is still presentable until then
		"22 December 2026",           // the civil track
		"section 142(1)(b)",
		"Section 147 makes the offence compoundable",
		Disclaimer,
	} {
		if !has(out, want) {
			t.Errorf("the checklist does not contain %q", want)
		}
	}
	if n := strings.Count(out, "[ ]"); n < 10 {
		t.Errorf("the checklist has %d tick boxes, want at least 10", n)
	}
}

func TestEveryDocumentCarriesTheDisclaimerAndSaysNothingWasSent(t *testing.T) {
	c := testCase(t)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	asOf := mustDay(t, "2024-02-01")

	docs := map[string]string{
		"demand notice":         DemandNotice(testOffice(), c, cas, asOf),
		"deadline schedule":     DeadlineSchedule(testOffice(), c, cas, asOf),
		"complaint particulars": ComplaintParticulars(testOffice(), c, cas, asOf),
		"case file checklist":   AnnexureChecklist(testOffice(), c, cas, asOf),
	}
	for name, out := range docs {
		if !has(out, Disclaimer) {
			t.Errorf("the %s does not carry the disclaimer", name)
		}
		if !has(out, NotSentNotice) {
			t.Errorf("the %s does not say that nothing has been sent", name)
		}
		if !has(out, "not legal advice") {
			t.Errorf("the %s must say it is not legal advice", name)
		}
		if !has(out, "confirm those periods") {
			t.Errorf("the %s must tell the reader to confirm the periods with an advocate", name)
		}
		if strings.TrimSpace(out) == "" {
			t.Errorf("the %s is empty", name)
		}
	}
}

const sampleParagraph = "Section 142(1)(b) gives one calendar month from the date on which the cause of action arises under clause (c) of the proviso to section 138."

func TestWrapFoldsWithoutLosingWords(t *testing.T) {
	got := wrap(sampleParagraph, 72)
	for _, line := range strings.Split(got, "\n") {
		if n := len([]rune(line)); n > 72 {
			t.Errorf("wrap produced a %d-character line: %q", n, line)
		}
	}
	if want, have := strings.Fields(sampleParagraph), strings.Fields(got); len(have) != len(want) {
		t.Errorf("wrap changed the word count: %d, want %d", len(have), len(want))
	}
	if got := wrap("", 72); got != "" {
		t.Errorf("wrapping nothing should produce nothing, got %q", got)
	}
	if got := wrap("   ", 72); got != "" {
		t.Errorf("wrapping blanks should produce nothing, got %q", got)
	}
	// A single word longer than the width still has to come out whole: a
	// cheque number or a case citation cut in half is worse than a long line.
	long := strings.Repeat("x", 100)
	if got := wrap(long, 72); got != long {
		t.Errorf("a word longer than the width must not be cut: got %q", got)
	}
}

// TestHangDoesNotRepeatTheBullet is the fix for a real defect: prefixing every
// folded line put "- " and "3." into the middle of sentences.
func TestHangDoesNotRepeatTheBullet(t *testing.T) {
	for _, prefix := range []string{"", "    ", "  - ", "3.  "} {
		got := hang(prefix, sampleParagraph, 72)
		lines := strings.Split(got, "\n")
		if len(lines) < 2 {
			t.Fatalf("hang(%q) did not fold the paragraph", prefix)
		}
		if !strings.HasPrefix(lines[0], prefix) {
			t.Errorf("hang(%q) first line = %q, want the prefix", prefix, lines[0])
		}
		want := strings.Repeat(" ", len([]rune(prefix)))
		for _, line := range lines[1:] {
			if !strings.HasPrefix(line, want) {
				t.Errorf("hang(%q) continuation %q is not indented to match", prefix, line)
			}
			if trimmed := strings.TrimSpace(prefix); trimmed != "" && strings.HasPrefix(strings.TrimSpace(line), trimmed) {
				t.Errorf("hang(%q) repeated the prefix on a continuation line: %q", prefix, line)
			}
			if n := len([]rune(line)); n > 72 {
				t.Errorf("hang(%q) produced a %d-character line", prefix, n)
			}
		}
		if want, have := flat(prefix+sampleParagraph), flat(got); have != want {
			t.Errorf("hang(%q) changed the text:\n got %q\nwant %q", prefix, have, want)
		}
	}
	if got := hang("  - ", "   ", 72); got != "" {
		t.Errorf("an empty item should print nothing, not a stray bullet; got %q", got)
	}
}

func TestAddressLinesSplitsAndFallsBack(t *testing.T) {
	got := addressLines("22 Industrial Suburb, Yeshwanthpur, Bengaluru 560022")
	want := []string{"22 Industrial Suburb,", "Yeshwanthpur,", "Bengaluru 560022"}
	if len(got) != len(want) {
		t.Fatalf("addressLines = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("addressLines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if got := addressLines("  "); len(got) != 1 || !strings.HasPrefix(got[0], "[ the drawer's full address") {
		t.Errorf("a blank address should print as a blank to fill, got %q", got)
	}
}

func TestOrFallsBackOnBlanks(t *testing.T) {
	cases := []struct{ value, fallback, want string }{
		{"something", "nothing", "something"},
		{"", "nothing", "nothing"},
		{"   ", "nothing", "nothing"},
	}
	for _, c := range cases {
		if got := Or(c.value, c.fallback); got != c.want {
			t.Errorf("Or(%q, %q) = %q, want %q", c.value, c.fallback, got, c.want)
		}
	}
}

func TestDaysPhrase(t *testing.T) {
	cases := []struct {
		days int
		want string
	}{
		{-10, "10 days ago"},
		{-1, "yesterday"},
		{0, "today"},
		{1, "tomorrow"},
		{28, "in 28 days"},
	}
	for _, c := range cases {
		if got := daysPhrase(c.days); got != c.want {
			t.Errorf("daysPhrase(%d) = %q, want %q", c.days, got, c.want)
		}
	}
}
