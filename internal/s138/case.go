package s138

import (
	"fmt"
	"strings"
)

// Office is the user's own details — the advocate or the firm keeping the
// docket. They go at the top of every document the tool prints.
type Office struct {
	Advocate  string `json:"advocate"`
	Firm      string `json:"firm"`
	Address   string `json:"address"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Enrolment string `json:"enrolment"`
}

// Tidy trims the office details.
func (o Office) Tidy() Office {
	o.Advocate = strings.TrimSpace(o.Advocate)
	o.Firm = strings.TrimSpace(o.Firm)
	o.Address = strings.TrimSpace(o.Address)
	o.Phone = strings.TrimSpace(o.Phone)
	o.Email = strings.TrimSpace(o.Email)
	o.Enrolment = strings.TrimSpace(o.Enrolment)
	return o
}

// Case is one dishonoured cheque and everything known about it. The dates live
// in the embedded Facts, which is what the cascade computes from; the rest is
// what the printed documents need.
type Case struct {
	ID  string `json:"id"`
	Ref string `json:"ref"`

	PayeeName    string `json:"payeeName"`
	PayeeAddress string `json:"payeeAddress"`

	DrawerName    string `json:"drawerName"`
	DrawerAddress string `json:"drawerAddress"`

	// Consideration is one line on what the cheque was given for — the
	// liability it was meant to discharge. Section 138 reaches a cheque drawn
	// "for the discharge, in whole or in part, of any debt or other
	// liability", so the notice and the complaint both have to say what that
	// debt was.
	Consideration string `json:"consideration"`

	ChequeNo string  `json:"chequeNo"`
	Amount   float64 `json:"amount"`

	DraweeBank   string `json:"draweeBank"`
	DraweeBranch string `json:"draweeBranch"`

	// CollectingBank is the payee's own bank, through which the cheque was
	// presented. Section 142(2)(a) puts jurisdiction with the court where the
	// branch of that bank is situated, where the cheque is delivered for
	// collection through an account.
	CollectingBank   string `json:"collectingBank"`
	CollectingBranch string `json:"collectingBranch"`

	MemoReason string `json:"memoReason"`
	NoticeMode string `json:"noticeMode"`

	CourtName string `json:"courtName"`
	CaseNo    string `json:"caseNo"`
	Notes     string `json:"notes"`

	Facts
}

// Tidy trims the free-text fields and rounds the amount to paise.
func (c Case) Tidy() Case {
	c.Ref = strings.TrimSpace(c.Ref)
	c.PayeeName = strings.TrimSpace(c.PayeeName)
	c.PayeeAddress = strings.TrimSpace(c.PayeeAddress)
	c.DrawerName = strings.TrimSpace(c.DrawerName)
	c.DrawerAddress = strings.TrimSpace(c.DrawerAddress)
	c.Consideration = strings.TrimSpace(c.Consideration)
	c.ChequeNo = strings.TrimSpace(c.ChequeNo)
	c.DraweeBank = strings.TrimSpace(c.DraweeBank)
	c.DraweeBranch = strings.TrimSpace(c.DraweeBranch)
	c.CollectingBank = strings.TrimSpace(c.CollectingBank)
	c.CollectingBranch = strings.TrimSpace(c.CollectingBranch)
	c.MemoReason = strings.TrimSpace(c.MemoReason)
	c.NoticeMode = strings.TrimSpace(c.NoticeMode)
	c.CourtName = strings.TrimSpace(c.CourtName)
	c.CaseNo = strings.TrimSpace(c.CaseNo)
	c.Notes = strings.TrimSpace(c.Notes)
	c.Amount = Round2(c.Amount)
	return c
}

// maxChequeAmount is a sanity bound on the typed figure, not a statutory one:
// a thousand crore in a cheque-bounce docket is a typo.
const maxChequeAmount = 1e13

// Validate checks what the law needs and what the arithmetic needs, and
// rejects date sequences that cannot have happened. A date entered in the
// wrong order would otherwise compute a deadline that looks authoritative and
// is wrong, which is the one failure this tool cannot afford.
func (c Case) Validate() error {
	switch {
	case c.Ref == "":
		return fmt.Errorf("give the case a file reference of your own, so you can find it again")
	case c.DrawerName == "":
		return fmt.Errorf("enter the name of the drawer — the person or company that signed the cheque")
	case c.ChequeNo == "":
		return fmt.Errorf("enter the cheque number")
	case c.Amount <= 0:
		return fmt.Errorf("enter the amount of the cheque in rupees")
	case c.Amount > maxChequeAmount:
		return fmt.Errorf("that cheque amount looks like a typing error — check it")
	case c.MemoDate.IsZero():
		return fmt.Errorf("enter the date on the bank's return memo: every deadline in the case is counted from it")
	}

	order := []struct {
		earlier, later Day
		msg            string
	}{
		{c.ChequeDate, c.PresentedDate, "the cheque cannot have been presented before it was drawn"},
		{c.ChequeDate, c.MemoDate, "the bank's return memo cannot be dated before the cheque"},
		{c.PresentedDate, c.MemoDate, "the bank's return memo cannot be dated before the cheque was presented"},
		{c.MemoDate, c.MemoReceivedDate, "you cannot have received the bank's information before the memo was written"},
		{c.MemoDate, c.NoticeSentDate, "the demand notice cannot have gone out before the cheque was returned"},
		{c.NoticeSentDate, c.NoticeServedDate, "the drawer cannot have received the notice before it was despatched"},
		{c.NoticeSentDate, c.ComplaintFiledDate, "the complaint cannot have been filed before the demand notice went out"},
	}
	for _, o := range order {
		if o.earlier.IsZero() || o.later.IsZero() {
			continue
		}
		if o.later.Before(o.earlier) {
			return fmt.Errorf("check the dates: %s", o.msg)
		}
	}

	for _, d := range []struct {
		name string
		day  Day
	}{
		{"cheque date", c.ChequeDate},
		{"presentation date", c.PresentedDate},
		{"memo date", c.MemoDate},
		{"date the bank's information was received", c.MemoReceivedDate},
		{"date the notice was despatched", c.NoticeSentDate},
		{"date the notice was served", c.NoticeServedDate},
		{"date the drawer paid", c.PaidDate},
		{"date the complaint was filed", c.ComplaintFiledDate},
	} {
		if d.day.IsZero() {
			continue
		}
		if y := d.day.Time().Year(); y < 1900 || y > 2200 {
			return fmt.Errorf("the %s (%s) is not a plausible date", d.name, d.day.String())
		}
	}
	return nil
}

// Cascade computes this case's position on the given date.
func (c Case) Cascade(p Periods, asOf Day) Cascade { return Compute(c.Facts, p, asOf) }

// Title is how the case reads in a list.
func (c Case) Title() string {
	ref := c.Ref
	if ref == "" {
		ref = "(no reference)"
	}
	return fmt.Sprintf("%s — %s, cheque %s", ref, c.DrawerName, c.ChequeNo)
}

// Or returns value, or the fallback when value is blank. Printed documents use
// it so that a field the user has not filled in reads as an obvious blank to
// complete rather than as nothing at all.
func Or(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
