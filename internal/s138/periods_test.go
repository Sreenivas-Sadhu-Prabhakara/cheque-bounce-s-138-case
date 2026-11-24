package s138

import (
	"strings"
	"testing"
)

// TestDefaultPeriodsAreTheEnactedFigures pins the constants themselves. If one
// of these changes, it must be because the law changed and someone meant it.
func TestDefaultPeriodsAreTheEnactedFigures(t *testing.T) {
	p := DefaultPeriods()
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"demand notice, s.138 proviso (b)", p.NoticeDays, 30},
		{"drawer's time to pay, s.138 proviso (c)", p.PaymentDays, 15},
		{"complaint, s.142(1)(b), in calendar months", p.ComplaintMonths, 1},
		{"cheque validity, RBI directions", p.ChequeValidityMonths, 3},
		{"presentation, s.138 proviso (a)", p.StatutoryPresentationMonths, 6},
		{"assumed days in the post (not statutory)", p.DeemedServiceDays, 7},
		{"civil suit, Limitation Act 1963", p.CivilSuitYears, 3},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
	if !p.IsStatutory() {
		t.Error("the default periods must report themselves as statutory")
	}
}

func TestNormaliseNeverLeavesAPeriodAtZero(t *testing.T) {
	cases := []struct {
		name string
		in   Periods
		want Periods
	}{
		{"everything unset", Periods{}, DefaultPeriods()},
		{
			"one field set, the rest filled from the Act",
			Periods{NoticeDays: 15},
			Periods{15, 15, 1, 3, 6, 7, 3},
		},
		{
			"negative values are not periods either",
			Periods{NoticeDays: -5, PaymentDays: -1, DeemedServiceDays: -9},
			DefaultPeriods(),
		},
		{"fully set is left alone", Periods{45, 20, 2, 12, 12, 10, 5}, Periods{45, 20, 2, 12, 12, 10, 5}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.in.Normalise(); got != c.want {
				t.Errorf("Normalise() = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestValidateRejectsImpossiblePeriods(t *testing.T) {
	cases := []struct {
		name    string
		in      Periods
		wantErr bool
	}{
		{"the statutory figures", DefaultPeriods(), false},
		{"a historic fifteen-day notice period", Periods{15, 15, 1, 3, 6, 7, 3}, false},
		{"zero normalises before validation", Periods{}.Normalise(), false},
		{"a notice period of a day", Periods{1, 15, 1, 3, 6, 7, 3}, false},
		{"a notice period of zero", Periods{0, 15, 1, 3, 6, 7, 3}, true},
		{"a notice period of two years", Periods{730, 15, 1, 3, 6, 7, 3}, true},
		{"no time at all to pay", Periods{30, 0, 1, 3, 6, 7, 3}, true},
		{"no complaint window", Periods{30, 15, 0, 3, 6, 7, 3}, true},
		{"a hundred-year complaint window", Periods{30, 15, 1200, 3, 6, 7, 3}, true},
		{"negative days in the post", Periods{30, 15, 1, 3, 6, -1, 3}, true},
		{"a century of civil limitation", Periods{30, 15, 1, 3, 6, 7, 100}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.in.Validate()
			if c.wantErr && err == nil {
				t.Errorf("Validate(%+v) accepted an impossible period", c.in)
			}
			if !c.wantErr && err != nil {
				t.Errorf("Validate(%+v) = %v, want no error", c.in, err)
			}
		})
	}
}

func TestIsStatutoryDetectsAnOverride(t *testing.T) {
	p := DefaultPeriods()
	p.ComplaintMonths = 2
	if p.IsStatutory() {
		t.Error("a changed complaint period must not report as statutory")
	}
	if !(Periods{}).Normalise().IsStatutory() {
		t.Error("the normalised zero value is the statutory set")
	}
}

// TestPeriodLinesStateEverySourceOnce makes sure the printed documents and the
// UI are describing the same figures the arithmetic ran on, with the provision
// each comes from, and that the one non-statutory assumption says so.
func TestPeriodLinesStateEverySourceOnce(t *testing.T) {
	lines := PeriodLines(DefaultPeriods())
	if len(lines) != 7 {
		t.Fatalf("PeriodLines returned %d lines, want one per period (7)", len(lines))
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"30 days", "s.138 proviso (b)",
		"15 days", "s.138 proviso (c)",
		"1 calendar month", "s.142(1)(b)", "General Clauses Act",
		"3 months", "Reserve Bank of India",
		"6 months", "s.138 proviso (a)",
		"NOT a statutory period",
		"3 years", "Limitation Act, 1963",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("PeriodLines does not mention %q:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "30 days  —  s.142") {
		t.Error("the complaint period must never be stated in days")
	}

	// An overridden figure has to appear as the figure actually used.
	p := DefaultPeriods()
	p.NoticeDays = 15
	if got := strings.Join(PeriodLines(p), "\n"); !strings.Contains(got, "15 days  —  s.138 proviso (b)") {
		t.Errorf("an overridden notice period must be stated as used:\n%s", got)
	}
}

func TestWarningDaysIsAPresentationThreshold(t *testing.T) {
	if WarningDays <= 0 || WarningDays > 30 {
		t.Errorf("WarningDays = %d; it should be a few days, not a statutory period", WarningDays)
	}
}

func TestPenaltyConstants(t *testing.T) {
	if PunishmentYears != 2 {
		t.Errorf("s.138 imprisonment limit = %d years, want 2", PunishmentYears)
	}
	if FineMultipleOfCheque != 2 {
		t.Errorf("s.138 fine limit = %d times the cheque, want 2", FineMultipleOfCheque)
	}
	if InterimCompensationMaxPct != 20 {
		t.Errorf("s.143A interim compensation ceiling = %d%%, want 20%%", InterimCompensationMaxPct)
	}
	if AppealDepositMinPct != 20 {
		t.Errorf("s.148 appeal deposit minimum = %d%%, want 20%%", AppealDepositMinPct)
	}
}
