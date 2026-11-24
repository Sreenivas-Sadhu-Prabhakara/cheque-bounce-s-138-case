package s138

import "fmt"

// The statutory periods, each with the provision it comes from. These are the
// only places in this tool where any of these numbers appears: every deadline
// the cascade reports is derived from them.
//
// They are the figures in force at the time of writing. They have moved
// before — the demand-notice period in clause (b) was fifteen days until the
// Negotiable Instruments (Amendment and Miscellaneous Provisions) Act, 2002
// raised it to thirty — and they can move again. Each one is overridable in
// the tool. Confirm the current figures, and your own facts, with an advocate
// before you rely on a date this tool prints.
const (
	// NoticeDays is the period within which the payee or holder in due course
	// must make the demand for payment in writing.
	//
	// Section 138, proviso (b), Negotiable Instruments Act, 1881: the payee
	// "makes a demand for the payment of the said amount of money by giving a
	// notice in writing, to the drawer of the cheque, within thirty days of
	// the receipt of information by him from the bank regarding the return of
	// the cheque as unpaid".
	//
	// Note what the period runs from: the receipt of the bank's information,
	// not the date on the return memo. Those are often the same day and
	// sometimes are not, so the cascade computes both and leads with the
	// earlier deadline.
	NoticeDays = 30

	// PaymentDays is the period the drawer has to pay after receiving the
	// demand notice. The offence is complete only when it expires unpaid.
	//
	// Section 138, proviso (c): the drawer "fails to make the payment of the
	// said amount of money to the payee or to the holder in due course of the
	// cheque within fifteen days of the receipt of the said notice".
	PaymentDays = 15

	// ComplaintMonths is the limitation period for filing the complaint,
	// counted in calendar months and not in days.
	//
	// Section 142(1)(b): "such complaint is made within one month of the date
	// on which the cause of action arises under clause (c) of the proviso to
	// section 138". Section 3(35) of the General Clauses Act, 1897 makes that
	// a month reckoned by the British calendar — 28, 29, 30 or 31 days
	// depending on the month, which is why it cannot be treated as 30 days.
	//
	// Treating it as 30 days is unsafe in exactly the case that matters: a
	// cause of action arising on 1 February gives a complaint deadline of
	// 1 March, while "30 days" would compute 3 March and file two days out of
	// time.
	ComplaintMonths = 1

	// ChequeValidityMonths is how long a cheque stays payable, and therefore
	// the practical limit on presenting it at all.
	//
	// Section 138, proviso (a) requires presentation "within a period of six
	// months from the date on which it is drawn or within the period of its
	// validity, whichever is earlier". Reserve Bank of India directions
	// effective 1 April 2012 reduced the validity of a cheque, draft, pay
	// order or banker's cheque to three months from the date of the
	// instrument, so for an ordinary cheque the operative limb is the validity
	// period, not the six months.
	ChequeValidityMonths = 3

	// StatutoryPresentationMonths is the six-month outer limb of proviso (a)
	// itself, kept separate from the validity period because the proviso takes
	// whichever of the two expires first.
	StatutoryPresentationMonths = 6

	// DeemedServiceDays is NOT a statutory period. It is the working
	// assumption this tool uses for how long a notice sent by registered post
	// or speed post takes to reach the drawer, when the actual date of receipt
	// is not yet known.
	//
	// Service matters because proviso (c) runs from "the receipt of the said
	// notice". Section 27 of the General Clauses Act, 1897 raises a
	// presumption of service where a notice is properly addressed, prepaid and
	// posted, and the drawer may rebut it. Until the acknowledgement or the
	// tracking record comes back, any date used here is a guess — so the
	// cascade also reports the deadline that assumes the drawer received the
	// notice on the day it was despatched, which is the earliest the window
	// can possibly close.
	DeemedServiceDays = 7

	// CivilSuitYears is the ordinary limitation period for a money suit on the
	// cheque — the parallel civil track, typically a summary suit under Order
	// XXXVII of the Code of Civil Procedure, 1908.
	//
	// Limitation Act, 1963, Schedule, Part II (suits relating to bills of
	// exchange and promissory notes): three years. It is much longer than the
	// criminal windows, which is why it is worth having on the same page: a
	// missed section 138 window does not necessarily mean a lost debt. The
	// article that applies turns on the instrument and the pleading — confirm
	// it with your advocate.
	CivilSuitYears = 3

	// PunishmentYears and FineMultipleOfCheque are the sentence under section
	// 138: "imprisonment for a term which may be extended to two years, or
	// with fine which may extend to twice the amount of the cheque, or with
	// both". They appear in the demand notice.
	PunishmentYears      = 2
	FineMultipleOfCheque = 2

	// InterimCompensationMaxPct is the ceiling on interim compensation the
	// trial court may order the drawer to pay during the trial: "not exceeding
	// twenty per cent of the amount of the cheque" (section 143A, inserted by
	// the Negotiable Instruments (Amendment) Act, 2018).
	InterimCompensationMaxPct = 20

	// AppealDepositMinPct is the minimum deposit an appellate court is to
	// order on an appeal against conviction: "a sum which shall be a minimum
	// of twenty per cent of the fine or compensation awarded" (section 148,
	// inserted in 2018).
	AppealDepositMinPct = 20
)

// WarningDays is how far ahead of a deadline this tool starts calling it
// urgent. It is a presentation threshold, not a legal one.
const WarningDays = 7

// Periods holds the periods a computation runs on. The defaults are the
// statutory figures above; a user who needs to work to a different figure —
// because the law moved, or because the instrument carried a validity period
// of its own — can override them, and every deadline recomputes.
type Periods struct {
	NoticeDays                  int `json:"noticeDays"`
	PaymentDays                 int `json:"paymentDays"`
	ComplaintMonths             int `json:"complaintMonths"`
	ChequeValidityMonths        int `json:"chequeValidityMonths"`
	StatutoryPresentationMonths int `json:"statutoryPresentationMonths"`
	DeemedServiceDays           int `json:"deemedServiceDays"`
	CivilSuitYears              int `json:"civilSuitYears"`
}

// DefaultPeriods returns the statutory periods as enacted.
func DefaultPeriods() Periods {
	return Periods{
		NoticeDays:                  NoticeDays,
		PaymentDays:                 PaymentDays,
		ComplaintMonths:             ComplaintMonths,
		ChequeValidityMonths:        ChequeValidityMonths,
		StatutoryPresentationMonths: StatutoryPresentationMonths,
		DeemedServiceDays:           DeemedServiceDays,
		CivilSuitYears:              CivilSuitYears,
	}
}

// Normalise fills in any period left at zero with the statutory default, so a
// partially written data file or a partially filled form still computes against
// the real rule rather than against zero.
//
// Every period here, DeemedServiceDays included, treats zero as "not set"
// rather than as a figure. A data file written by an older version, or a form
// field left blank, must never silently reduce a statutory window to nothing.
// The same-day-service case that a zero would express is already covered: the
// cascade always reports the deadline computed on service the day the notice
// was despatched, as the date to work to.
func (p Periods) Normalise() Periods {
	d := DefaultPeriods()
	if p.NoticeDays <= 0 {
		p.NoticeDays = d.NoticeDays
	}
	if p.PaymentDays <= 0 {
		p.PaymentDays = d.PaymentDays
	}
	if p.ComplaintMonths <= 0 {
		p.ComplaintMonths = d.ComplaintMonths
	}
	if p.ChequeValidityMonths <= 0 {
		p.ChequeValidityMonths = d.ChequeValidityMonths
	}
	if p.StatutoryPresentationMonths <= 0 {
		p.StatutoryPresentationMonths = d.StatutoryPresentationMonths
	}
	if p.DeemedServiceDays <= 0 {
		p.DeemedServiceDays = d.DeemedServiceDays
	}
	if p.CivilSuitYears <= 0 {
		p.CivilSuitYears = d.CivilSuitYears
	}
	return p
}

// Validate rejects overrides that could not be a limitation period, so a typo
// in the settings form cannot quietly move every deadline in the docket.
func (p Periods) Validate() error {
	checks := []struct {
		name     string
		value    int
		min, max int
	}{
		{"demand-notice period (days)", p.NoticeDays, 1, 365},
		{"drawer's payment period (days)", p.PaymentDays, 1, 365},
		{"complaint period (months)", p.ComplaintMonths, 1, 60},
		{"cheque validity (months)", p.ChequeValidityMonths, 1, 60},
		{"presentation limb of proviso (a) (months)", p.StatutoryPresentationMonths, 1, 120},
		{"assumed days in the post", p.DeemedServiceDays, 1, 90},
		{"civil suit limitation (years)", p.CivilSuitYears, 1, 30},
	}
	for _, c := range checks {
		if c.value < c.min || c.value > c.max {
			return fmt.Errorf("the %s must be between %d and %d, not %d", c.name, c.min, c.max, c.value)
		}
	}
	return nil
}

// IsStatutory reports whether the periods in use are the enacted figures, so
// the UI and every printed document can say plainly when a deadline was
// computed on a figure the user changed.
func (p Periods) IsStatutory() bool { return p.Normalise() == DefaultPeriods() }
