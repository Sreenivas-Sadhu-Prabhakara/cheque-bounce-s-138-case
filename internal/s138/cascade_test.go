package s138

import (
	"reflect"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// The individual deadline functions
// ---------------------------------------------------------------------------

func TestNoticeDeadline(t *testing.T) {
	cases := []struct {
		name string
		from string
		want string
	}{
		{"memo on 31 January in a leap year", "2024-01-31", "2024-03-01"},
		{"memo on 31 January in an ordinary year", "2023-01-31", "2023-03-02"},
		{"memo on 1 February in a leap year", "2024-02-01", "2024-03-02"},
		{"memo on 1 February in an ordinary year", "2023-02-01", "2023-03-03"},
		{"memo on 15 December runs into the new year", "2024-12-15", "2025-01-14"},
		{"memo on 30 November", "2024-11-30", "2024-12-30"},
		{"memo on 29 February", "2024-02-29", "2024-03-30"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NoticeDeadline(mustDay(t, c.from), DefaultPeriods())
			if got.String() != c.want {
				t.Errorf("NoticeDeadline(%s) = %s, want %s", c.from, got, c.want)
			}
		})
	}
}

func TestPaymentDeadlineAndCauseOfAction(t *testing.T) {
	cases := []struct {
		name          string
		served        string
		wantPayBy     string
		wantCauseOfAc string
	}{
		// Saketh India Ltd v. India Securities Ltd: notice received 15 June
		// 1996, the fifteen days expire on 30 June, the cause of action arises
		// on 1 July.
		{"Saketh facts", "1996-06-15", "1996-06-30", "1996-07-01"},
		{"served 16 Jan, expiry lands on 31 Jan", "2024-01-16", "2024-01-31", "2024-02-01"},
		{"served 14 Feb in a leap year", "2024-02-14", "2024-02-29", "2024-03-01"},
		{"served 14 Feb in an ordinary year", "2023-02-14", "2023-03-01", "2023-03-02"},
		{"served 31 Dec runs into the new year", "2024-12-31", "2025-01-15", "2025-01-16"},
		{"served 15 Jan", "2024-01-15", "2024-01-30", "2024-01-31"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			served := mustDay(t, c.served)
			if got := PaymentDeadline(served, DefaultPeriods()); got.String() != c.wantPayBy {
				t.Errorf("PaymentDeadline(%s) = %s, want %s", c.served, got, c.wantPayBy)
			}
			if got := CauseOfActionDate(served, DefaultPeriods()); got.String() != c.wantCauseOfAc {
				t.Errorf("CauseOfActionDate(%s) = %s, want %s", c.served, got, c.wantCauseOfAc)
			}
		})
	}
}

func TestComplaintDeadlines(t *testing.T) {
	cases := []struct {
		name                string
		served              string
		wantSafe, wantOuter string
	}{
		// Saketh again: the complaint filed on 31 July 1996 was held in time,
		// which the outer limit below allows and the safe date does not.
		{"Saketh facts", "1996-06-15", "1996-07-30", "1996-08-01"},

		// The month-end clamp, in a leap year and out of one. These are the
		// cases a naive +1 month would push into March and lose the case on.
		{"expiry on 31 Jan, leap year", "2024-01-16", "2024-02-29", "2024-03-01"},
		{"expiry on 31 Jan, ordinary year", "2023-01-16", "2023-02-28", "2023-03-01"},
		{"expiry on 30 Jan, leap year: both clamp to 29 Feb", "2024-01-15", "2024-02-29", "2024-02-29"},
		{"expiry on 30 Jan, ordinary year: both clamp to 28 Feb", "2023-01-15", "2023-02-28", "2023-02-28"},
		{"expiry on 29 Jan, leap year", "2024-01-14", "2024-02-29", "2024-02-29"},
		{"expiry on 31 Mar", "2024-03-16", "2024-04-30", "2024-05-01"},
		{"ordinary month", "2024-11-24", "2025-01-09", "2025-01-10"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			safe, outer := ComplaintDeadlines(mustDay(t, c.served), DefaultPeriods())
			if safe.String() != c.wantSafe {
				t.Errorf("safe complaint deadline from service on %s = %s, want %s", c.served, safe, c.wantSafe)
			}
			if outer.String() != c.wantOuter {
				t.Errorf("outer complaint deadline from service on %s = %s, want %s", c.served, outer, c.wantOuter)
			}
			if outer.Before(safe) {
				t.Errorf("the outer limit %s cannot fall before the safe date %s", outer, safe)
			}
		})
	}
}

// TestComplaintMonthIsNotThirtyDays is the test for the one conflict between
// the brief this tool was built from and the Act. Section 142(1)(b) gives "one
// month", not thirty days, and in a short month the month is the shorter of
// the two: working to thirty days would file out of time.
func TestComplaintMonthIsNotThirtyDays(t *testing.T) {
	// Notice served on 16 January 2023: the fifteen days expire on 31 January
	// and the cause of action arises on 1 February 2023.
	served := mustDay(t, "2023-01-16")
	p := DefaultPeriods()

	coa := CauseOfActionDate(served, p)
	if got, want := coa.String(), "2023-02-01"; got != want {
		t.Fatalf("cause of action = %s, want %s", got, want)
	}

	_, outer := ComplaintDeadlines(served, p)
	if got, want := outer.String(), "2023-03-01"; got != want {
		t.Errorf("one month from %s = %s, want %s", coa, got, want)
	}

	naive := coa.AddDays(30)
	if got, want := naive.String(), "2023-03-03"; got != want {
		t.Fatalf("thirty days from %s = %s, want %s", coa, got, want)
	}
	if !outer.Before(naive) {
		t.Errorf("one month (%s) must fall before thirty days (%s) off a 1 February cause of action", outer, naive)
	}
	if days := outer.DaysUntil(naive); days != 2 {
		t.Errorf("thirty days overshoots the month by %d days, want 2", days)
	}
}

func TestLastPresentationDate(t *testing.T) {
	cases := []struct {
		name                     string
		cheque                   string
		periods                  Periods
		wantEffective, wantOuter string
	}{
		{"30 Nov into a short February", "2024-11-30", DefaultPeriods(), "2025-02-28", "2025-05-30"},
		{"30 Nov into a leap February", "2023-11-30", DefaultPeriods(), "2024-02-29", "2024-05-30"},
		{"31 Jan", "2024-01-31", DefaultPeriods(), "2024-04-30", "2024-07-31"},
		{"31 Dec", "2023-12-31", DefaultPeriods(), "2024-03-31", "2024-06-30"},
		{"29 Feb", "2024-02-29", DefaultPeriods(), "2024-05-29", "2024-08-29"},
		// An instrument whose own validity is longer than the six months in
		// the proviso: the proviso's six months then becomes the earlier limb.
		{"validity overridden past the proviso", "2024-01-31",
			Periods{ChequeValidityMonths: 12}, "2024-07-31", "2024-07-31"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			effective, outer := LastPresentationDate(mustDay(t, c.cheque), c.periods)
			if effective.String() != c.wantEffective {
				t.Errorf("effective last presentation date = %s, want %s", effective, c.wantEffective)
			}
			if outer.String() != c.wantOuter {
				t.Errorf("six-month limb = %s, want %s", outer, c.wantOuter)
			}
		})
	}
}

func TestCivilSuitDeadlineClampsTheLeapDay(t *testing.T) {
	cases := []struct{ memo, want string }{
		{"2024-02-29", "2027-02-28"},
		{"2024-01-31", "2027-01-31"},
		{"2023-11-23", "2026-11-23"},
	}
	for _, c := range cases {
		if got := CivilSuitDeadline(mustDay(t, c.memo), DefaultPeriods()); got.String() != c.want {
			t.Errorf("CivilSuitDeadline(%s) = %s, want %s", c.memo, got, c.want)
		}
	}
}

func TestDeemedServiceAndEarliestComplaintDeadline(t *testing.T) {
	p := DefaultPeriods()
	sent := mustDay(t, "2024-01-16")

	if got, want := DeemedServiceDate(sent, p).String(), "2024-01-23"; got != want {
		t.Errorf("deemed service = %s, want %s", got, want)
	}
	// Same-day service is the earliest the window can close: 16 Jan + 15 days
	// is 31 Jan, and one month from there clamps to 29 February.
	if got, want := EarliestComplaintDeadline(sent, p).String(), "2024-02-29"; got != want {
		t.Errorf("earliest complaint deadline = %s, want %s", got, want)
	}
	// It must never fall later than the deadline computed on assumed service.
	safe, _ := ComplaintDeadlines(DeemedServiceDate(sent, p), p)
	if EarliestComplaintDeadline(sent, p).After(safe) {
		t.Error("the earliest possible deadline cannot fall after the expected one")
	}
}

// ---------------------------------------------------------------------------
// Compute: the window boundaries, a day either side of each one
// ---------------------------------------------------------------------------

// baseFacts is a clean case: cheque drawn 1 December 2023, presented and
// dishonoured in time, notice out on 10 January 2024 and served on 16 January.
// The fifteen days then expire on 31 January 2024, the cause of action arises
// on 1 February, and the complaint is due by 29 February.
func baseFacts(t *testing.T) Facts {
	t.Helper()
	return Facts{
		ChequeDate:       mustDay(t, "2023-12-01"),
		PresentedDate:    mustDay(t, "2023-12-20"),
		MemoDate:         mustDay(t, "2023-12-22"),
		NoticeSentDate:   mustDay(t, "2024-01-10"),
		NoticeServedDate: mustDay(t, "2024-01-16"),
	}
}

func stepOf(t *testing.T, c Cascade, key string) Step {
	t.Helper()
	s, ok := c.Step(key)
	if !ok {
		t.Fatalf("cascade has no %q step", key)
	}
	return s
}

func TestComputeBaseCaseDates(t *testing.T) {
	c := Compute(baseFacts(t), DefaultPeriods(), mustDay(t, "2024-02-01"))

	want := map[string]string{
		"present":   "2024-03-01", // 1 Dec + 3 months validity
		"notice":    "2024-01-21", // 22 Dec + 30 days
		"pay":       "2024-01-31", // 16 Jan + 15 days
		"complaint": "2024-02-29", // one month from 31 Jan, clamped
		"civil":     "2026-12-22", // 22 Dec + 3 years
	}
	for key, date := range want {
		if got := stepOf(t, c, key).Due; got.String() != date {
			t.Errorf("%s deadline = %s, want %s", key, got, date)
		}
	}
	if got, want := c.CauseOfAction.String(), "2024-02-01"; got != want {
		t.Errorf("cause of action = %s, want %s", got, want)
	}
	if got, want := c.FileBy.String(), "2024-02-29"; got != want {
		t.Errorf("date to file by = %s, want %s", got, want)
	}
	if got, want := stepOf(t, c, "complaint").Outer.String(), "2024-03-01"; got != want {
		t.Errorf("outer complaint limit = %s, want %s", got, want)
	}
	if got, want := c.ServiceBasis, "recorded"; got != want {
		t.Errorf("service basis = %q, want %q", got, want)
	}
	if len(c.Fatal) != 0 {
		t.Errorf("a clean case should have no claim-ending problems, got %v", c.Fatal)
	}
	if !c.Alive {
		t.Error("a clean case should be alive")
	}
	if got, want := c.Stage, StageComplaint; got != want {
		t.Errorf("stage = %q, want %q", got, want)
	}
}

func TestComputeNoticeWindowBoundaries(t *testing.T) {
	// Memo dated 31 January 2024, so the thirty days expire on 1 March 2024.
	cases := []struct {
		name      string
		asOf      string
		sent      string
		wantState StepState
		wantFatal bool
	}{
		{"the day before, not yet sent", "2024-02-29", "", StateDue, false},
		{"the deadline itself, not yet sent", "2024-03-01", "", StateDue, false},
		{"the day after, never sent", "2024-03-02", "", StateMissed, true},
		{"a week later, never sent", "2024-03-08", "", StateMissed, true},
		{"sent well inside the window", "2024-03-05", "2024-02-10", StateDone, false},
		{"sent on the deadline itself", "2024-03-05", "2024-03-01", StateDone, false},
		{"sent the day after the deadline", "2024-03-05", "2024-03-02", StateMissed, true},
		{"sent two months late", "2024-05-05", "2024-04-30", StateMissed, true},
		{"not yet sent, three weeks to run", "2024-02-10", "", StatePending, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := Facts{MemoDate: mustDay(t, "2024-01-31")}
			if c.sent != "" {
				f.NoticeSentDate = mustDay(t, c.sent)
			}
			got := Compute(f, DefaultPeriods(), mustDay(t, c.asOf))
			step := stepOf(t, got, "notice")
			if step.Due.String() != "2024-03-01" {
				t.Fatalf("notice deadline = %s, want 2024-03-01", step.Due)
			}
			if step.State != c.wantState {
				t.Errorf("notice state = %q, want %q", step.State, c.wantState)
			}
			if hasFatal := len(got.Fatal) > 0; hasFatal != c.wantFatal {
				t.Errorf("claim-ending problem = %v, want %v (%v)", hasFatal, c.wantFatal, got.Fatal)
			}
		})
	}
}

func TestComputeNoticeWindowRunsFromTheReceiptOfTheBankInformation(t *testing.T) {
	// Memo dated 31 January, information received on 5 February. The safe
	// deadline is 1 March (memo + 30); the arguable one is 6 March.
	f := Facts{
		MemoDate:         mustDay(t, "2024-01-31"),
		MemoReceivedDate: mustDay(t, "2024-02-05"),
	}

	c := Compute(f, DefaultPeriods(), mustDay(t, "2024-02-10"))
	step := stepOf(t, c, "notice")
	if got, want := step.Due.String(), "2024-03-01"; got != want {
		t.Errorf("safe notice deadline = %s, want %s", got, want)
	}
	if got, want := step.Outer.String(), "2024-03-06"; got != want {
		t.Errorf("outer notice deadline = %s, want %s", got, want)
	}

	// A notice sent between the two is a warning, not a dead claim.
	f.NoticeSentDate = mustDay(t, "2024-03-03")
	c = Compute(f, DefaultPeriods(), mustDay(t, "2024-03-10"))
	if got := stepOf(t, c, "notice").State; got != StateLate {
		t.Errorf("a notice sent inside the outer window is %q, want %q", got, StateLate)
	}
	if len(c.Fatal) != 0 {
		t.Errorf("that should not be a claim-ending problem, got %v", c.Fatal)
	}
	if len(c.Warnings) == 0 {
		t.Error("that should raise a warning")
	}

	// Past the outer date with nothing sent, the claim is gone.
	f.NoticeSentDate = Day{}
	c = Compute(f, DefaultPeriods(), mustDay(t, "2024-03-07"))
	if got := stepOf(t, c, "notice").State; got != StateMissed {
		t.Errorf("past the outer window the notice step is %q, want %q", got, StateMissed)
	}
	if len(c.Fatal) == 0 {
		t.Error("past the outer window the claim is extinguished and must say so")
	}
}

func TestComputePaymentWindowBoundaries(t *testing.T) {
	// Notice served 16 January 2024, so the fifteen days expire on 31 January.
	cases := []struct {
		name      string
		asOf      string
		paid      string
		wantState StepState
		wantStage Stage
	}{
		{"mid-window", "2024-01-20", "", StateWaiting, StageWaiting},
		{"the last day", "2024-01-31", "", StateWaiting, StageWaiting},
		{"the day after", "2024-02-01", "", StateElapsed, StageComplaint},
		{"paid on the first day of the window", "2024-02-10", "2024-01-17", StateDone, StagePaid},
		{"paid on the last day of the window", "2024-02-10", "2024-01-31", StateDone, StagePaid},
		{"paid the day after the window closed", "2024-02-10", "2024-02-01", StateElapsed, StageSettledLate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := baseFacts(t)
			if c.paid != "" {
				f.PaidDate = mustDay(t, c.paid)
			}
			got := Compute(f, DefaultPeriods(), mustDay(t, c.asOf))
			step := stepOf(t, got, "pay")
			if step.Due.String() != "2024-01-31" {
				t.Fatalf("payment deadline = %s, want 2024-01-31", step.Due)
			}
			if step.State != c.wantState {
				t.Errorf("payment state = %q, want %q", step.State, c.wantState)
			}
			if got.Stage != c.wantStage {
				t.Errorf("stage = %q, want %q", got.Stage, c.wantStage)
			}
			if step.Actor != "The drawer" {
				t.Errorf("the payment period belongs to the drawer, not to %q", step.Actor)
			}
		})
	}
}

func TestComputeComplaintWindowBoundaries(t *testing.T) {
	// Base case: safe deadline 29 February 2024, outer limit 1 March 2024.
	cases := []struct {
		name      string
		asOf      string
		filed     string
		wantState StepState
		wantFatal bool
		wantWarn  bool
	}{
		{"three weeks to run", "2024-02-05", "", StatePending, false, false},
		{"the day before the safe date", "2024-02-28", "", StateDue, false, false},
		{"the safe date itself", "2024-02-29", "", StateDue, false, false},
		{"the day after the safe date, inside the outer limit", "2024-03-01", "", StateDue, false, true},
		{"the day after the outer limit", "2024-03-02", "", StateMissed, true, false},
		{"filed inside the window", "2024-03-10", "2024-02-20", StateDone, false, false},
		{"filed on the safe date", "2024-03-10", "2024-02-29", StateDone, false, false},
		{"filed on the outer limit", "2024-03-10", "2024-03-01", StateDone, false, true},
		{"filed after the outer limit", "2024-03-10", "2024-03-02", StateMissed, true, false},
		{"filed before the cause of action arose", "2024-02-10", "2024-01-25", StateLate, false, true},
		{"filed on the day the cause of action arose", "2024-02-10", "2024-02-01", StateDone, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := baseFacts(t)
			if c.filed != "" {
				f.ComplaintFiledDate = mustDay(t, c.filed)
			}
			got := Compute(f, DefaultPeriods(), mustDay(t, c.asOf))
			step := stepOf(t, got, "complaint")
			if step.Due.String() != "2024-02-29" || step.Outer.String() != "2024-03-01" {
				t.Fatalf("complaint window = %s / %s, want 2024-02-29 / 2024-03-01", step.Due, step.Outer)
			}
			if step.State != c.wantState {
				t.Errorf("complaint state = %q, want %q", step.State, c.wantState)
			}
			if hasFatal := len(got.Fatal) > 0; hasFatal != c.wantFatal {
				t.Errorf("claim-ending problem = %v, want %v (%v)", hasFatal, c.wantFatal, got.Fatal)
			}
			if hasWarn := len(got.Warnings) > 0; hasWarn != c.wantWarn {
				t.Errorf("warning = %v, want %v (%v)", hasWarn, c.wantWarn, got.Warnings)
			}
		})
	}
}

// TestComputeUnprovedServiceUsesTheEarliestPossibleDeadline is the safety rule
// of the whole tool: while service is only assumed, the window can close
// sooner than expected, so the date to work to is the earliest it can close.
func TestComputeUnprovedServiceUsesTheEarliestPossibleDeadline(t *testing.T) {
	f := baseFacts(t)
	f.NoticeServedDate = Day{} // the acknowledgement is still in the post

	c := Compute(f, DefaultPeriods(), mustDay(t, "2024-01-20"))

	if got, want := c.ServiceBasis, "deemed"; got != want {
		t.Fatalf("service basis = %q, want %q", got, want)
	}
	// Notice despatched 10 January + 7 days in the post = 17 January.
	if got, want := c.ServiceDate.String(), "2024-01-17"; got != want {
		t.Errorf("deemed service date = %s, want %s", got, want)
	}
	step := stepOf(t, c, "complaint")
	// On assumed service: 17 Jan + 15 = 1 Feb, one month = 1 March.
	if got, want := step.Due.String(), "2024-03-01"; got != want {
		t.Errorf("expected complaint deadline = %s, want %s", got, want)
	}
	// On same-day service: 10 Jan + 15 = 25 Jan, one month = 25 February.
	if got, want := c.FileBy.String(), "2024-02-25"; got != want {
		t.Errorf("date to file by = %s, want %s — it must assume the earliest service", got, want)
	}
	if !c.FileBy.Before(step.Due) {
		t.Error("the date to work to must be earlier than the date computed on assumed service")
	}
	if !strings.Contains(step.Note, "25 February 2024") {
		t.Errorf("the complaint step should explain the earlier date; note was %q", step.Note)
	}
	if !strings.Contains(c.NextAction, "25 February 2024") {
		t.Errorf("the next action should name the date to file by; got %q", c.NextAction)
	}

	// Once service is proved, the tool stops guessing and uses the real date.
	f.NoticeServedDate = mustDay(t, "2024-01-16")
	c = Compute(f, DefaultPeriods(), mustDay(t, "2024-01-20"))
	if got, want := c.ServiceBasis, "recorded"; got != want {
		t.Errorf("service basis = %q, want %q", got, want)
	}
	if got, want := c.FileBy.String(), "2024-02-29"; got != want {
		t.Errorf("date to file by = %s, want %s", got, want)
	}
}

func TestComputePresentationWindowBoundaries(t *testing.T) {
	// Cheque dated 31 January 2024: payable to 30 April 2024 on the
	// three-month validity, and 31 July 2024 on the six-month limb.
	cases := []struct {
		name      string
		presented string
		memo      string
		wantState StepState
		wantFatal bool
		wantWarn  bool
	}{
		{"presented at once", "2024-02-01", "2024-02-03", StateDone, false, false},
		{"presented on the last day of validity", "2024-04-30", "2024-05-02", StateDone, false, false},
		{"presented the day after validity ran out", "2024-05-01", "2024-05-03", StateMissed, true, false},
		{"presented after the six-month limb too", "2024-08-01", "2024-08-03", StateMissed, true, false},
		// No presentation date recorded: the memo date stands in, and a memo
		// dated after the validity is a warning to go and find the real date,
		// not a finding that the claim is dead.
		{"no presentation date, memo after validity", "", "2024-05-01", StateDue, false, true},
		{"no presentation date, memo inside validity", "", "2024-04-29", StateDone, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// The notice goes out the day after each memo, and the position is
			// read five days later, so the only step that can raise a problem
			// in this table is the presentation check itself.
			memo := mustDay(t, c.memo)
			f := Facts{
				ChequeDate:     mustDay(t, "2024-01-31"),
				MemoDate:       memo,
				NoticeSentDate: memo.AddDays(1),
			}
			if c.presented != "" {
				f.PresentedDate = mustDay(t, c.presented)
			}
			got := Compute(f, DefaultPeriods(), memo.AddDays(5))
			step := stepOf(t, got, "present")
			if step.Due.String() != "2024-04-30" || step.Outer.String() != "2024-07-31" {
				t.Fatalf("presentation window = %s / %s, want 2024-04-30 / 2024-07-31", step.Due, step.Outer)
			}
			if step.State != c.wantState {
				t.Errorf("presentation state = %q, want %q", step.State, c.wantState)
			}
			if hasFatal := len(got.Fatal) > 0; hasFatal != c.wantFatal {
				t.Errorf("claim-ending problem = %v, want %v (%v)", hasFatal, c.wantFatal, got.Fatal)
			}
			if hasWarn := len(got.Warnings) > 0; hasWarn != c.wantWarn {
				t.Errorf("warning = %v, want %v (%v)", hasWarn, c.wantWarn, got.Warnings)
			}
		})
	}
}

func TestComputeWithoutAChequeDateWarnsRatherThanGuesses(t *testing.T) {
	f := Facts{MemoDate: mustDay(t, "2024-01-31")}
	c := Compute(f, DefaultPeriods(), mustDay(t, "2024-02-05"))

	step := stepOf(t, c, "present")
	if step.State != StateUnknown {
		t.Errorf("without a cheque date the presentation check is %q, want %q", step.State, StateUnknown)
	}
	if !step.Due.IsZero() {
		t.Errorf("without a cheque date there can be no presentation deadline, got %s", step.Due)
	}
	if len(c.Warnings) == 0 {
		t.Error("a missing cheque date must be flagged: proviso (a) is a precondition to the offence")
	}
}

func TestComputeWithoutAMemoDateComputesNothing(t *testing.T) {
	c := Compute(Facts{}, DefaultPeriods(), mustDay(t, "2024-02-05"))

	if got, want := c.Stage, StageNoDishonour; got != want {
		t.Errorf("stage = %q, want %q", got, want)
	}
	if len(c.Steps) != 0 {
		t.Errorf("with no dishonour there is nothing to compute, got %d steps", len(c.Steps))
	}
	if c.Alive {
		t.Error("a case with no dishonour on record is not a live case")
	}
	if !strings.Contains(c.NextAction, "return memo") {
		t.Errorf("the next action should ask for the memo date, got %q", c.NextAction)
	}
}

func TestComputeStagesBeforeTheNoticeGoesOut(t *testing.T) {
	f := Facts{
		ChequeDate: mustDay(t, "2023-12-01"),
		MemoDate:   mustDay(t, "2023-12-22"),
	}
	c := Compute(f, DefaultPeriods(), mustDay(t, "2023-12-26"))

	if got, want := c.Stage, StageNotice; got != want {
		t.Errorf("stage = %q, want %q", got, want)
	}
	if got, want := c.ServiceBasis, "unknown"; got != want {
		t.Errorf("service basis = %q, want %q", got, want)
	}
	for _, key := range []string{"pay", "complaint"} {
		if got := stepOf(t, c, key).State; got != StateUnknown {
			t.Errorf("%s step is %q before the notice goes out, want %q", key, got, StateUnknown)
		}
	}
	if got, want := c.NextKey, "notice"; got != want {
		t.Errorf("next step = %q, want %q", got, want)
	}
	if got, want := c.NextDue.String(), "2024-01-21"; got != want {
		t.Errorf("next deadline = %s, want %s", got, want)
	}
}

func TestComputeNextActionFollowsTheCascade(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*testing.T, *Facts)
		asOf    string
		wantKey string
	}{
		{"nothing done yet", func(t *testing.T, f *Facts) { f.NoticeSentDate, f.NoticeServedDate = Day{}, Day{} }, "2023-12-26", "notice"},
		{"notice served, the drawer's window running", func(t *testing.T, f *Facts) {}, "2024-01-20", "pay"},
		{"window elapsed, complaint to file", func(t *testing.T, f *Facts) {}, "2024-02-05", "complaint"},
		{
			"complaint filed: nothing left on the section 138 clock",
			func(t *testing.T, f *Facts) { f.ComplaintFiledDate = mustDay(t, "2024-02-20") },
			"2024-03-05", "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := baseFacts(t)
			c.mutate(t, &f)
			got := Compute(f, DefaultPeriods(), mustDay(t, c.asOf))
			if got.NextKey != c.wantKey {
				t.Errorf("next step = %q, want %q", got.NextKey, c.wantKey)
			}
			if got.NextAction == "" {
				t.Error("there should always be a next action to show")
			}
		})
	}
}

// TestComputePaidInTimeLeavesNoComplaintToFile guards a defect found by
// driving the binary: a cheque paid inside the fifteen days was still being
// told to file a complaint. Clause (c) is not satisfied, so there is no
// offence, no complaint and no limitation period — and a complaint deadline
// shown on a paid cheque reads as a live date.
func TestComputePaidInTimeLeavesNoComplaintToFile(t *testing.T) {
	f := baseFacts(t)
	f.PaidDate = mustDay(t, "2024-01-25") // the fifteen days expire on 31 January
	c := Compute(f, DefaultPeriods(), mustDay(t, "2024-02-20"))

	step := stepOf(t, c, "complaint")
	if step.State != StateNA {
		t.Errorf("complaint state = %q, want %q", step.State, StateNA)
	}
	if !step.Due.IsZero() || !step.Outer.IsZero() {
		t.Errorf("a paid cheque has no complaint deadline, got %s / %s", step.Due, step.Outer)
	}
	if !c.FileBy.IsZero() {
		t.Errorf("a paid cheque has no date to file by, got %s", c.FileBy)
	}
	if c.NextKey != "" {
		t.Errorf("next step = %q, want nothing on the clock", c.NextKey)
	}
	if !strings.Contains(c.NextAction, "No offence arises") {
		t.Errorf("next action = %q, want it to say no offence arises", c.NextAction)
	}
	if got, want := c.Stage, StagePaid; got != want {
		t.Errorf("stage = %q, want %q", got, want)
	}
	if c.Alive {
		t.Error("a paid case is not live")
	}

	// Paid AFTER the window is the opposite: the offence was complete, so the
	// complaint and its limitation period are very much alive.
	f.PaidDate = mustDay(t, "2024-02-01")
	c = Compute(f, DefaultPeriods(), mustDay(t, "2024-02-20"))
	step = stepOf(t, c, "complaint")
	if step.State == StateNA {
		t.Error("paying after the window does not extinguish the complaint")
	}
	if got, want := step.Due.String(), "2024-02-29"; got != want {
		t.Errorf("complaint deadline = %s, want %s", got, want)
	}
	if got, want := c.Stage, StageSettledLate; got != want {
		t.Errorf("stage = %q, want %q", got, want)
	}
}

func TestComputeAfterFilingHasNothingLeftOnTheClock(t *testing.T) {
	f := baseFacts(t)
	f.ComplaintFiledDate = mustDay(t, "2024-02-20")
	c := Compute(f, DefaultPeriods(), mustDay(t, "2024-03-10"))

	if got, want := c.Stage, StageFiled; got != want {
		t.Errorf("stage = %q, want %q", got, want)
	}
	if !strings.Contains(c.NextAction, "court's dates") {
		t.Errorf("after filing the next action should hand over to the court, got %q", c.NextAction)
	}
}

func TestComputeHonoursOverriddenPeriods(t *testing.T) {
	// The pre-2002 figure for proviso (b) was fifteen days. A user working a
	// historic file can set it, and every date must move with it.
	p := DefaultPeriods()
	p.NoticeDays = 15

	c := Compute(Facts{MemoDate: mustDay(t, "2024-01-31")}, p, mustDay(t, "2024-02-05"))
	if got, want := stepOf(t, c, "notice").Due.String(), "2024-02-15"; got != want {
		t.Errorf("notice deadline on a fifteen-day period = %s, want %s", got, want)
	}
	if c.Statutory {
		t.Error("a cascade computed on changed periods must not claim to be statutory")
	}

	// And a zero-valued Periods must fall back to the Act, never to zero.
	c = Compute(baseFacts(t), Periods{}, mustDay(t, "2024-02-01"))
	if !c.Statutory {
		t.Error("an empty Periods should normalise to the statutory figures")
	}
	if got, want := stepOf(t, c, "complaint").Due.String(), "2024-02-29"; got != want {
		t.Errorf("complaint deadline on normalised periods = %s, want %s", got, want)
	}
}

func TestComputeIsPure(t *testing.T) {
	f := baseFacts(t)
	asOf := mustDay(t, "2024-02-01")
	first := Compute(f, DefaultPeriods(), asOf)
	second := Compute(f, DefaultPeriods(), asOf)
	if !reflect.DeepEqual(first, second) {
		t.Error("Compute must give the same answer for the same facts and date")
	}
}

func TestComputeFlagsWeekendDeadlines(t *testing.T) {
	// Memo dated 6 December 2023 puts the notice deadline on 5 January 2024, a
	// Friday; memo dated 7 December puts it on Saturday 6 January.
	weekday := Compute(Facts{MemoDate: mustDay(t, "2023-12-06")}, DefaultPeriods(), mustDay(t, "2023-12-10"))
	if s := stepOf(t, weekday, "notice"); s.Weekend {
		t.Errorf("%s is a %s and should not be flagged as a weekend", s.Due, s.Due.Weekday())
	}
	weekend := Compute(Facts{MemoDate: mustDay(t, "2023-12-07")}, DefaultPeriods(), mustDay(t, "2023-12-10"))
	s := stepOf(t, weekend, "notice")
	if got, want := s.Due.String(), "2024-01-06"; got != want {
		t.Fatalf("notice deadline = %s, want %s", got, want)
	}
	if !s.Weekend {
		t.Errorf("%s is a %s and should be flagged", s.Due, s.Due.Weekday())
	}
}

func TestComputeEveryStepCarriesItsSectionAndRule(t *testing.T) {
	c := Compute(baseFacts(t), DefaultPeriods(), mustDay(t, "2024-02-01"))
	wantSections := map[string]string{
		"present":   "s.138 proviso (a)",
		"notice":    "s.138 proviso (b)",
		"pay":       "s.138 proviso (c)",
		"complaint": "s.142(1)(b)",
		"civil":     "Limitation Act, 1963",
	}
	if len(c.Steps) != len(wantSections) {
		t.Fatalf("cascade has %d steps, want %d", len(c.Steps), len(wantSections))
	}
	for _, s := range c.Steps {
		if want := wantSections[s.Key]; s.Section != want {
			t.Errorf("%s step cites %q, want %q", s.Key, s.Section, want)
		}
		if s.Rule == "" {
			t.Errorf("%s step has no stated rule", s.Key)
		}
		if s.Name == "" || s.Action == "" || s.Actor == "" {
			t.Errorf("%s step is missing its name, action or actor", s.Key)
		}
	}
}

func TestComputeDaysLeftCountsFromAsOf(t *testing.T) {
	cases := []struct {
		asOf string
		want int
	}{
		{"2024-02-01", 28},
		{"2024-02-28", 1},
		{"2024-02-29", 0},
		{"2024-03-01", -1},
	}
	for _, c := range cases {
		got := Compute(baseFacts(t), DefaultPeriods(), mustDay(t, c.asOf))
		if days := stepOf(t, got, "complaint").DaysLeft; days != c.want {
			t.Errorf("on %s the complaint is %d days away, want %d", c.asOf, days, c.want)
		}
	}
}
