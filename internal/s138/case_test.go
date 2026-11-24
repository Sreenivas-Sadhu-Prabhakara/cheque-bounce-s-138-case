package s138

import (
	"strings"
	"testing"
)

func TestValidateRequiresWhatTheCascadeNeeds(t *testing.T) {
	valid := testCase(t)

	cases := []struct {
		name    string
		mutate  func(*Case)
		wantErr string
	}{
		{"a complete case", func(c *Case) {}, ""},
		{"no file reference", func(c *Case) { c.Ref = "" }, "file reference"},
		{"a blank file reference", func(c *Case) { c.Ref = "   " }, "file reference"},
		{"no drawer", func(c *Case) { c.DrawerName = "" }, "drawer"},
		{"no cheque number", func(c *Case) { c.ChequeNo = "" }, "cheque number"},
		{"no amount", func(c *Case) { c.Amount = 0 }, "amount of the cheque"},
		{"a negative amount", func(c *Case) { c.Amount = -1 }, "amount of the cheque"},
		{"an absurd amount", func(c *Case) { c.Amount = 1e14 }, "typing error"},
		{"no memo date", func(c *Case) { c.MemoDate = Day{} }, "return memo"},
		// Only the memo date is indispensable; a case can be recorded the day
		// the memo arrives, before anything else is known.
		{"nothing but the essentials", func(c *Case) {
			*c = Case{Ref: "X/1", DrawerName: "D", ChequeNo: "1", Amount: 100,
				Facts: Facts{MemoDate: mustDay(t, "2024-01-05")}}
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			subject := valid
			c.mutate(&subject)
			err := subject.Tidy().Validate()
			switch {
			case c.wantErr == "" && err != nil:
				t.Errorf("Validate() = %v, want no error", err)
			case c.wantErr != "" && err == nil:
				t.Errorf("Validate() accepted the case, want an error about %q", c.wantErr)
			case c.wantErr != "" && !strings.Contains(err.Error(), c.wantErr):
				t.Errorf("Validate() = %v, want an error about %q", err, c.wantErr)
			}
		})
	}
}

// TestValidateRejectsImpossibleDateSequences is the guard that matters most: a
// date keyed in the wrong order would produce a deadline that looks
// authoritative and is wrong.
func TestValidateRejectsImpossibleDateSequences(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Case)
		reject bool
	}{
		{"the real sequence", func(c *Case) {}, false},
		{"presented before the cheque was drawn", func(c *Case) {
			c.PresentedDate = mustDay(t, "2023-11-30")
		}, true},
		{"memo dated before the cheque", func(c *Case) {
			c.ChequeDate = mustDay(t, "2024-01-01")
		}, true},
		{"memo dated before presentation", func(c *Case) {
			c.PresentedDate = mustDay(t, "2023-12-23")
		}, true},
		{"the bank's information received before the memo", func(c *Case) {
			c.MemoReceivedDate = mustDay(t, "2023-12-20")
		}, true},
		{"notice sent before the cheque was returned", func(c *Case) {
			c.NoticeSentDate = mustDay(t, "2023-12-20")
		}, true},
		{"notice served before it was sent", func(c *Case) {
			c.NoticeServedDate = mustDay(t, "2024-01-09")
		}, true},
		{"notice served on the day it was sent", func(c *Case) {
			c.NoticeServedDate = c.NoticeSentDate
		}, false},
		{"complaint filed before the notice went out", func(c *Case) {
			c.ComplaintFiledDate = mustDay(t, "2024-01-05")
		}, true},
		{"a date in the wrong century", func(c *Case) {
			c.MemoDate = mustDay(t, "1823-12-22")
		}, true},
		{"a date far in the future", func(c *Case) {
			c.PaidDate = mustDay(t, "2301-01-01")
		}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			subject := testCase(t)
			c.mutate(&subject)
			err := subject.Tidy().Validate()
			if c.reject && err == nil {
				t.Error("Validate() accepted an impossible sequence of dates")
			}
			if !c.reject && err != nil {
				t.Errorf("Validate() = %v, want no error", err)
			}
		})
	}
}

func TestTidyTrimsAndRounds(t *testing.T) {
	in := Case{
		Ref:        "  NI/2024/07  ",
		DrawerName: "  Veda Fabricators  ",
		ChequeNo:   " 004512 ",
		MemoReason: "  Funds insufficient  ",
		Notes:      "  chase the guarantor  ",
		Amount:     250000.005,
	}
	got := in.Tidy()
	if got.Ref != "NI/2024/07" || got.DrawerName != "Veda Fabricators" || got.ChequeNo != "004512" {
		t.Errorf("Tidy() left whitespace: %+v", got)
	}
	if got.MemoReason != "Funds insufficient" || got.Notes != "chase the guarantor" {
		t.Errorf("Tidy() left whitespace in the free text: %+v", got)
	}
	if got.Amount != Round2(250000.005) {
		t.Errorf("Tidy() amount = %v, want %v", got.Amount, Round2(250000.005))
	}
}

func TestOfficeTidy(t *testing.T) {
	got := Office{Advocate: "  A. N. Rao ", Enrolment: " KAR/1234/2009 ", Email: " a@b.invalid "}.Tidy()
	if got.Advocate != "A. N. Rao" || got.Enrolment != "KAR/1234/2009" || got.Email != "a@b.invalid" {
		t.Errorf("Office.Tidy() left whitespace: %+v", got)
	}
}

func TestCaseTitle(t *testing.T) {
	c := testCase(t)
	if got, want := c.Title(), "NI/2024/07 — Veda Fabricators Private Limited, cheque 004512"; got != want {
		t.Errorf("Title() = %q, want %q", got, want)
	}
	if got := (Case{}).Title(); !strings.Contains(got, "(no reference)") {
		t.Errorf("an unreferenced case should say so, got %q", got)
	}
}

func TestCaseCascadeUsesItsOwnFacts(t *testing.T) {
	c := testCase(t)
	cas := c.Cascade(DefaultPeriods(), mustDay(t, "2024-02-01"))
	direct := Compute(c.Facts, DefaultPeriods(), mustDay(t, "2024-02-01"))
	if cas.FileBy != direct.FileBy || cas.Stage != direct.Stage {
		t.Error("Case.Cascade must compute from the case's own facts")
	}
}
