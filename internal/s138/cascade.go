package s138

import "fmt"

// ---------------------------------------------------------------------------
// The individual deadlines, as pure functions of one date and the periods.
//
// Each one is a single line so that the rule it encodes can be read against
// the section it comes from. Nothing else in the tool does this arithmetic.
// ---------------------------------------------------------------------------

// LastPresentationDate returns the last day the cheque can be presented to the
// bank, and the six-month outer limb of proviso (a) beside it.
//
// Section 138, proviso (a): presentation "within a period of six months from
// the date on which it is drawn or within the period of its validity,
// whichever is earlier". Since the Reserve Bank reduced cheque validity to
// three months in 2012, the earlier of the two is normally the validity.
func LastPresentationDate(chequeDate Day, p Periods) (effective, statutory Day) {
	p = p.Normalise()
	validity := chequeDate.AddMonths(p.ChequeValidityMonths)
	statutory = chequeDate.AddMonths(p.StatutoryPresentationMonths)
	return Earlier(validity, statutory), statutory
}

// NoticeDeadline returns the last day to give the demand notice, counted from
// the day the payee received the bank's information about the return.
//
// Section 138, proviso (b): within thirty days of that receipt. The day of
// receipt is excluded — section 9 of the General Clauses Act, 1897 provides
// that "from" a day excludes that day — so the window is the thirty days that
// follow it and the last day is receipt + 30.
func NoticeDeadline(infoReceived Day, p Periods) Day {
	return infoReceived.AddDays(p.Normalise().NoticeDays)
}

// PaymentDeadline returns the last day the drawer may pay and avoid the
// offence, counted from the day the drawer received the demand notice.
//
// Section 138, proviso (c): fifteen days of the receipt of the notice, again
// excluding the day of receipt.
func PaymentDeadline(noticeServed Day, p Periods) Day {
	return noticeServed.AddDays(p.Normalise().PaymentDays)
}

// CauseOfActionDate returns the day the cause of action under clause (c) of
// the proviso arises: the day after the drawer's fifteen days expire unpaid.
// The offence is not complete until then, because until then the drawer has
// not yet failed to pay within the period.
//
// Econ Antri Ltd v. Rom Industries Ltd, (2014) 11 SCC 769, a three-judge bench
// affirming Saketh India Ltd v. India Securities Ltd, (1999) 3 SCC 1, puts it
// as the day immediately following the day on which the fifteen days expire.
func CauseOfActionDate(noticeServed Day, p Periods) Day {
	return PaymentDeadline(noticeServed, p).AddDays(1)
}

// ComplaintDeadlines returns the two defensible last days for filing the
// complaint under section 142(1)(b): the one to work to, and the outer limit.
//
//   - safe is one calendar month from the expiry of the drawer's fifteen days.
//   - outer is one calendar month from the date the cause of action arises,
//     which is the day after that expiry. This is the Econ Antri reckoning and
//     is the better view of the law.
//
// The two differ by a day, or by nothing at all when the month-end clamp
// catches both. This tool leads with safe and prints outer beside it: filing
// on the safe date is in time on either reckoning, and the one-day saving is
// never worth arguing about with a case in the balance.
func ComplaintDeadlines(noticeServed Day, p Periods) (safe, outer Day) {
	p = p.Normalise()
	expiry := PaymentDeadline(noticeServed, p)
	return expiry.AddMonths(p.ComplaintMonths), CauseOfActionDate(noticeServed, p).AddMonths(p.ComplaintMonths)
}

// EarliestComplaintDeadline returns the earliest date on which the complaint
// window can possibly close, computed on the assumption that the drawer
// received the notice on the very day it was despatched.
//
// This is the date to work to while the acknowledgement is still in the post.
// If the notice turns out to have been served sooner than assumed, the window
// closes sooner than the expected deadline — and a complaint filed by this
// date is in time whatever the court later finds about service.
func EarliestComplaintDeadline(noticeSent Day, p Periods) Day {
	safe, _ := ComplaintDeadlines(noticeSent, p)
	return safe
}

// DeemedServiceDate returns the date a notice despatched on sent is assumed to
// have reached the drawer. The assumption is not statutory; see
// DeemedServiceDays.
func DeemedServiceDate(sent Day, p Periods) Day {
	return sent.AddDays(p.Normalise().DeemedServiceDays)
}

// CivilSuitDeadline returns the last day for a money suit on the cheque,
// counted from the dishonour.
func CivilSuitDeadline(memoDate Day, p Periods) Day {
	return memoDate.AddYears(p.Normalise().CivilSuitYears)
}

// ---------------------------------------------------------------------------
// The cascade
// ---------------------------------------------------------------------------

// Facts are the dates the cascade is computed from. Only MemoDate is required;
// the rest are filled in as the case moves, and the cascade says what is
// unknown rather than guessing silently.
type Facts struct {
	// ChequeDate is the date written on the cheque.
	ChequeDate Day `json:"chequeDate"`
	// PresentedDate is the date the cheque was presented for collection.
	PresentedDate Day `json:"presentedDate"`
	// MemoDate is the date on the bank's return or dishonour memo.
	MemoDate Day `json:"memoDate"`
	// MemoReceivedDate is the date the payee received the bank's information
	// about the return. Proviso (b) runs from this date, not from MemoDate.
	MemoReceivedDate Day `json:"memoReceivedDate"`
	// NoticeSentDate is the date the demand notice was despatched.
	NoticeSentDate Day `json:"noticeSentDate"`
	// NoticeServedDate is the date the drawer received the demand notice, once
	// the acknowledgement or the tracking record proves it.
	NoticeServedDate Day `json:"noticeServedDate"`
	// PaidDate is the date the drawer paid the cheque amount in full.
	PaidDate Day `json:"paidDate"`
	// ComplaintFiledDate is the date the complaint was filed in court.
	ComplaintFiledDate Day `json:"complaintFiledDate"`
}

// StepState is where one step of the cascade stands.
type StepState string

const (
	// StatePending — the step is yours to do and there is time.
	StatePending StepState = "pending"
	// StateDue — the step is yours to do and the deadline is close or past.
	StateDue StepState = "due"
	// StateDone — the step was done in time.
	StateDone StepState = "done"
	// StateLate — the step was done, but after the date to work to.
	StateLate StepState = "late"
	// StateMissed — the window closed without the step being done.
	StateMissed StepState = "missed"
	// StateWaiting — the other side's clock is running; nothing to do yet.
	StateWaiting StepState = "waiting"
	// StateElapsed — the other side's period has run out without payment.
	StateElapsed StepState = "elapsed"
	// StateUnknown — a date this step depends on has not been recorded.
	StateUnknown StepState = "unknown"
	// StateNA — the step does not arise on these facts at all.
	StateNA StepState = "na"
)

// Step is one rung of the cascade, with the arithmetic that produced it.
type Step struct {
	Key      string    `json:"key"`
	Name     string    `json:"name"`
	Section  string    `json:"section"`
	Actor    string    `json:"actor"`
	From     Day       `json:"from"`
	FromWhat string    `json:"fromWhat"`
	Due      Day       `json:"due"`
	Outer    Day       `json:"outer"`
	Done     Day       `json:"done"`
	DaysLeft int       `json:"daysLeft"`
	State    StepState `json:"state"`
	Weekend  bool      `json:"weekend"`
	Rule     string    `json:"rule"`
	Action   string    `json:"action"`
	Note     string    `json:"note"`
}

// Stage is the single word for where the whole case stands.
type Stage string

const (
	StageNoDishonour Stage = "no-dishonour"
	StageNotice      Stage = "notice"
	StageWaiting     Stage = "waiting"
	StageComplaint   Stage = "complaint"
	StageFiled       Stage = "filed"
	StagePaid        Stage = "paid"
	StageSettledLate Stage = "settled-late"
	StageBarred      Stage = "barred"
)

var stageLabels = map[Stage]string{
	StageNoDishonour: "No dishonour recorded",
	StageNotice:      "Dishonoured — demand notice to go out",
	StageWaiting:     "Notice issued — the drawer's fifteen days are running",
	StageComplaint:   "Cause of action arisen — complaint to be filed",
	StageFiled:       "Complaint filed",
	StagePaid:        "Paid within the notice period — no offence",
	StageSettledLate: "Paid after the period expired",
	StageBarred:      "A statutory window has closed",
}

// Cascade is the whole computed position of one case on one date.
type Cascade struct {
	AsOf          Day      `json:"asOf"`
	Periods       Periods  `json:"periods"`
	Statutory     bool     `json:"statutory"`
	Stage         Stage    `json:"stage"`
	StageLabel    string   `json:"stageLabel"`
	Steps         []Step   `json:"steps"`
	NextKey       string   `json:"nextKey"`
	NextAction    string   `json:"nextAction"`
	NextDue       Day      `json:"nextDue"`
	NextDaysLeft  int      `json:"nextDaysLeft"`
	ServiceDate   Day      `json:"serviceDate"`
	ServiceBasis  string   `json:"serviceBasis"`
	CauseOfAction Day      `json:"causeOfAction"`
	FileBy        Day      `json:"fileBy"`
	Fatal         []string `json:"fatal"`
	Warnings      []string `json:"warnings"`
	Alive         bool     `json:"alive"`
}

// Step returns the step with the given key.
func (c Cascade) Step(key string) (Step, bool) {
	for _, s := range c.Steps {
		if s.Key == key {
			return s, true
		}
	}
	return Step{}, false
}

// Compute derives the whole deadline cascade from the facts on record, as at
// the given date. It is pure: same facts, same date, same answer.
func Compute(f Facts, p Periods, asOf Day) Cascade {
	p = p.Normalise()
	// The three lists start empty rather than nil so that the JSON always has
	// them as arrays: a caller reading "no problems" must not have to tell an
	// absent list from an empty one.
	c := Cascade{
		AsOf:      asOf,
		Periods:   p,
		Statutory: p.IsStatutory(),
		Alive:     true,
		Steps:     []Step{},
		Fatal:     []string{},
		Warnings:  []string{},
	}

	if f.MemoDate.IsZero() {
		c.Stage = StageNoDishonour
		c.StageLabel = stageLabels[StageNoDishonour]
		c.NextAction = "Record the date on the bank's return memo. Every deadline in the case is counted from it."
		c.Alive = false
		return c
	}

	// Which day did the payee learn of the return? Proviso (b) runs from the
	// receipt of the bank's information. Where that is not recorded the memo
	// date stands in, which is the earlier and therefore the safer of the two.
	infoReceived := f.MemoReceivedDate
	if infoReceived.IsZero() {
		infoReceived = f.MemoDate
	}

	c.addPresentationStep(f, p)
	c.addNoticeStep(f, p, asOf, infoReceived)
	service, basis := serviceDate(f, p)
	c.ServiceDate, c.ServiceBasis = service, basis
	c.addPaymentStep(f, p, asOf, service, basis)
	c.addComplaintStep(f, p, asOf, service, basis)
	c.addCivilStep(f, p, asOf)

	c.Stage = stageOf(f, p, asOf, service, len(c.Fatal) > 0)
	c.StageLabel = stageLabels[c.Stage]
	c.Alive = len(c.Fatal) == 0 && c.Stage != StagePaid
	c.setNext()
	return c
}

// serviceDate decides which date the drawer's fifteen days run from, and says
// on what basis.
func serviceDate(f Facts, p Periods) (Day, string) {
	switch {
	case !f.NoticeServedDate.IsZero():
		return f.NoticeServedDate, "recorded"
	case !f.NoticeSentDate.IsZero():
		return DeemedServiceDate(f.NoticeSentDate, p), "deemed"
	default:
		return Day{}, "unknown"
	}
}

func (c *Cascade) fatal(format string, args ...any) {
	c.Fatal = append(c.Fatal, fmt.Sprintf(format, args...))
}

func (c *Cascade) warn(format string, args ...any) {
	c.Warnings = append(c.Warnings, fmt.Sprintf(format, args...))
}

func (c *Cascade) add(s Step) {
	s.Weekend = !s.Due.IsZero() && s.Due.IsWeekend()
	if !s.Due.IsZero() && !c.AsOf.IsZero() {
		s.DaysLeft = c.AsOf.DaysUntil(s.Due)
	}
	if !s.Outer.IsZero() && s.Outer.Equal(s.Due) {
		s.Outer = Day{}
	}
	c.Steps = append(c.Steps, s)
}

// --- step 1: presentation -------------------------------------------------

func (c *Cascade) addPresentationStep(f Facts, p Periods) {
	step := Step{
		Key:      "present",
		Name:     "Cheque presented while still payable",
		Section:  "s.138 proviso (a)",
		Actor:    "You",
		From:     f.ChequeDate,
		FromWhat: "the date on the cheque",
		Rule: fmt.Sprintf("cheque date + %d months (validity), or + %d months under the proviso, whichever expires first",
			p.ChequeValidityMonths, p.StatutoryPresentationMonths),
		Action: "Present the cheque before its validity runs out.",
	}

	if f.ChequeDate.IsZero() {
		step.State = StateUnknown
		step.Note = "The date on the cheque has not been recorded, so this precondition cannot be checked."
		c.add(step)
		c.warn("Record the date written on the cheque. Without it the tool cannot check whether the cheque was presented while it was still payable, which section 138 proviso (a) requires before there can be any offence at all.")
		return
	}

	effective, statutory := LastPresentationDate(f.ChequeDate, p)
	step.Due, step.Outer = effective, statutory

	presented, exact := f.PresentedDate, true
	if presented.IsZero() {
		presented, exact = f.MemoDate, false
		step.Note = "The date of presentation has not been recorded, so the date on the return memo stands in for it. A memo is often dated a day or two after the cheque reached the bank."
	}
	step.Done = presented

	switch {
	case presented.After(statutory):
		step.State = StateMissed
		c.fatal("The cheque was presented on %s, after the six-month limb of section 138 proviso (a) expired on %s. There is no offence under section 138 on this presentation. A cheque that is still within its validity can be presented again, and a fresh dishonour gives a fresh cause of action.",
			presented.Pretty(), statutory.Pretty())
	case presented.After(effective) && exact:
		step.State = StateMissed
		c.fatal("The cheque was presented on %s, after it ceased to be payable on %s (%d months from the date on the cheque). Section 138 proviso (a) requires presentation within the period of the cheque's validity, so this presentation cannot support a complaint. If the instrument in fact carried a longer validity, change the validity period in the settings and recompute.",
			presented.Pretty(), effective.Pretty(), p.ChequeValidityMonths)
	case presented.After(effective):
		step.State = StateDue
		c.warn("On the dates recorded the cheque reached the bank on %s, after it ceased to be payable on %s. The date used is the memo date, because the date of presentation has not been recorded — record the actual presentation date, because section 138 proviso (a) turns on it.",
			presented.Pretty(), effective.Pretty())
	default:
		step.State = StateDone
	}
	c.add(step)
}

// --- step 2: the demand notice --------------------------------------------

func (c *Cascade) addNoticeStep(f Facts, p Periods, asOf, infoReceived Day) {
	safe := NoticeDeadline(f.MemoDate, p)
	outer := NoticeDeadline(infoReceived, p)

	step := Step{
		Key:      "notice",
		Name:     "Demand notice given to the drawer",
		Section:  "s.138 proviso (b)",
		Actor:    "You",
		From:     f.MemoDate,
		FromWhat: "the date on the bank's return memo",
		Due:      safe,
		Outer:    outer,
		Done:     f.NoticeSentDate,
		Rule:     fmt.Sprintf("memo date + %d days (the proviso runs from the receipt of the bank's information; the memo date is the earlier and safer start)", p.NoticeDays),
		Action:   "Issue the written demand for the cheque amount on the drawer, by registered post with acknowledgement due and by speed post, and keep the receipts.",
	}
	if !outer.Equal(safe) {
		step.Note = fmt.Sprintf("You have recorded the bank's information as received on %s, which would put the window out to %s. Working to %s instead means never having to prove when you opened the bank's letter.",
			infoReceived.Pretty(), outer.Pretty(), safe.Pretty())
	}

	sent := f.NoticeSentDate
	switch {
	case sent.IsZero() && asOf.After(outer):
		step.State = StateMissed
		c.fatal("The demand notice was never given, and the thirty-day window under section 138 proviso (b) closed on %s. No cause of action under section 138 can arise on this dishonour. If the cheque is still within its validity, presenting it again and serving a notice on the fresh dishonour starts the cascade over; the civil suit on the cheque is a separate track and runs much longer.",
			outer.Pretty())
	case sent.IsZero() && asOf.After(safe):
		step.State = StateDue
		c.warn("The demand notice is already past %s, the deadline counted from the memo date. It is arguably still open until %s, counted from the day you received the bank's information, but that puts the whole case on your being able to prove that later date. Issue the notice today.",
			safe.Pretty(), outer.Pretty())
	case sent.IsZero():
		step.State = StatePending
		if asOf.DaysUntil(safe) <= WarningDays {
			step.State = StateDue
		}
	case sent.After(outer):
		step.State = StateMissed
		c.fatal("The demand notice was despatched on %s, after the thirty-day window under section 138 proviso (b) closed on %s. A notice given out of time cannot found a complaint under section 138.",
			sent.Pretty(), outer.Pretty())
	case sent.After(safe):
		step.State = StateLate
		c.warn("The demand notice went out on %s, after %s — the thirty days counted from the memo date. It is within the window only if you can show you received the bank's information on %s or later, so keep whatever proves that date.",
			sent.Pretty(), safe.Pretty(), infoReceived.Pretty())
	default:
		step.State = StateDone
	}
	c.add(step)
}

// --- step 3: the drawer's fifteen days ------------------------------------

func (c *Cascade) addPaymentStep(f Facts, p Periods, asOf, service Day, basis string) {
	step := Step{
		Key:      "pay",
		Name:     "Drawer's fifteen days to pay",
		Section:  "s.138 proviso (c)",
		Actor:    "The drawer",
		From:     service,
		FromWhat: "the date the drawer received the notice",
		Rule:     fmt.Sprintf("date of receipt of the notice + %d days; the offence is complete the day after it expires unpaid", p.PaymentDays),
		Action:   "Nothing to do but wait and keep the proof of service safe.",
		Done:     f.PaidDate,
	}

	if basis == "unknown" {
		step.State = StateUnknown
		step.Note = "This period starts only when the drawer receives the notice, so it cannot be computed until the notice has gone out."
		c.add(step)
		return
	}

	due := PaymentDeadline(service, p)
	step.Due = due
	switch basis {
	case "deemed":
		step.Note = fmt.Sprintf("Service has not been proved yet. The date used is the despatch date plus %d days in the post, which is this tool's assumption and not a statutory period. Record the date on the acknowledgement or the tracking record as soon as it comes back.", p.DeemedServiceDays)
	case "recorded":
		step.Note = "Counted from the date of receipt you have recorded. Keep the acknowledgement or the tracking record on the file: this date fixes every date after it."
	}

	switch {
	case !f.PaidDate.IsZero() && f.PaidDate.OnOrBefore(due):
		step.State = StateDone
		step.Note = fmt.Sprintf("Paid on %s, within the fifteen days. Clause (c) of the proviso is not satisfied, so no offence under section 138 arises on this cheque. Close the file and keep the receipt.", f.PaidDate.Pretty())
	case !f.PaidDate.IsZero():
		step.State = StateElapsed
		step.Note = fmt.Sprintf("Paid on %s, after the fifteen days expired on %s. The offence was already complete, and payment afterwards does not by itself undo it — though an offence under section 138 is compoundable under section 147, and payment is what compounding usually turns on. Take advice before withdrawing anything.", f.PaidDate.Pretty(), due.Pretty())
	case asOf.After(due):
		step.State = StateElapsed
	default:
		step.State = StateWaiting
	}
	c.add(step)
}

// --- step 4: the complaint ------------------------------------------------

func (c *Cascade) addComplaintStep(f Facts, p Periods, asOf, service Day, basis string) {
	step := Step{
		Key:      "complaint",
		Name:     "Complaint filed before the magistrate",
		Section:  "s.142(1)(b)",
		Actor:    "You",
		FromWhat: "the date the cause of action arose",
		Rule:     fmt.Sprintf("one calendar month from the expiry of the drawer's %d days — a calendar month, not %d days", p.PaymentDays, 30),
		Action:   "File the written complaint, with the affidavit, before the Metropolitan Magistrate or Judicial Magistrate of the first class with jurisdiction.",
		Done:     f.ComplaintFiledDate,
	}

	if basis == "unknown" {
		step.State = StateUnknown
		step.Note = "This window opens only once the drawer's fifteen days have run, so it cannot be computed until the notice has gone out."
		c.add(step)
		return
	}

	// If the drawer paid inside the fifteen days, clause (c) of the proviso is
	// not satisfied and no offence arises: there is no complaint to file and no
	// limitation period running. Showing a complaint deadline here would be
	// worse than showing none, because it reads as a live date.
	if !f.PaidDate.IsZero() && f.PaidDate.OnOrBefore(PaymentDeadline(service, p)) {
		step.State = StateNA
		step.Action = "Nothing to file."
		step.Note = fmt.Sprintf("The drawer paid on %s, within the fifteen days. No cause of action arises under clause (c) of the proviso, so there is no complaint and no limitation period. Close the file.", f.PaidDate.Pretty())
		c.add(step)
		return
	}

	coa := CauseOfActionDate(service, p)
	safe, outer := ComplaintDeadlines(service, p)
	c.CauseOfAction = coa
	step.From, step.Due, step.Outer = coa, safe, outer

	// While service is only assumed, the window can close earlier than the
	// expected date. The earliest it can possibly close is the deadline
	// computed on same-day service, and that is the date to work to.
	fileBy := safe
	if basis != "recorded" && !f.NoticeSentDate.IsZero() {
		earliest := EarliestComplaintDeadline(f.NoticeSentDate, p)
		if earliest.Before(fileBy) {
			fileBy = earliest
			step.Note = fmt.Sprintf("Service is assumed, not proved, so this deadline is a guess. If the drawer in fact received the notice on the day it was despatched, the window closes on %s — %d days earlier. Work to %s until the acknowledgement proves otherwise.",
				earliest.Pretty(), earliest.DaysUntil(safe), earliest.Pretty())
		}
	}
	c.FileBy = fileBy
	if step.Note == "" && !outer.Equal(safe) {
		step.Note = fmt.Sprintf("The outer limit on the better view of the law is %s, one month from the cause of action itself. The one day between the two is not worth arguing about: file by %s.", outer.Pretty(), safe.Pretty())
	}

	filed := f.ComplaintFiledDate
	switch {
	case filed.IsZero() && asOf.After(outer):
		step.State = StateMissed
		c.fatal("The complaint was not filed and the one-month period under section 142(1)(b) closed on %s. The proviso to that clause lets a court take cognizance after the period if you satisfy it that there was sufficient cause for not filing in time — that is an indulgence to ask for, not a right to rely on. Take advice at once.",
			outer.Pretty())
	case filed.IsZero() && asOf.After(fileBy):
		step.State = StateDue
		c.warn("The complaint is past %s, the date to work to. The outer limit is %s. File now.", fileBy.Pretty(), outer.Pretty())
	case filed.IsZero():
		step.State = StatePending
		if asOf.DaysUntil(fileBy) <= WarningDays {
			step.State = StateDue
		}
	case filed.Before(coa):
		step.State = StateLate
		c.warn("The complaint is recorded as filed on %s, before the cause of action arose on %s. A complaint filed before the drawer's fifteen days have expired is premature and liable to be dismissed on that ground alone. Check the date.",
			filed.Pretty(), coa.Pretty())
	case filed.After(outer):
		step.State = StateMissed
		c.fatal("The complaint is recorded as filed on %s, after the one-month period under section 142(1)(b) closed on %s. Cognizance now depends on the court condoning the delay under the proviso to section 142(1)(b).",
			filed.Pretty(), outer.Pretty())
	case filed.After(safe):
		step.State = StateDone
		c.warn("The complaint was filed on %s, after %s but within the outer limit of %s. In time on the reckoning in Econ Antri Ltd v. Rom Industries Ltd, but be ready to argue the point.",
			filed.Pretty(), safe.Pretty(), outer.Pretty())
	default:
		step.State = StateDone
	}
	c.add(step)
}

// --- step 5: the civil track ----------------------------------------------

func (c *Cascade) addCivilStep(f Facts, p Periods, asOf Day) {
	due := CivilSuitDeadline(f.MemoDate, p)
	step := Step{
		Key:      "civil",
		Name:     "Money suit on the cheque (the parallel track)",
		Section:  "Limitation Act, 1963",
		Actor:    "You",
		From:     f.MemoDate,
		FromWhat: "the dishonour",
		Due:      due,
		Rule:     fmt.Sprintf("dishonour + %d years", p.CivilSuitYears),
		Action:   "A suit for the money, usually a summary suit under Order XXXVII of the Code of Civil Procedure, 1908.",
		Note:     "This track is here because it is the reason a missed section 138 window is not necessarily a lost debt: it runs for years, not weeks. The criminal complaint and the civil suit are separate remedies and may both be pursued. The article of the Limitation Act that applies turns on the instrument and the pleading — settle it with your advocate.",
	}
	switch {
	case !f.PaidDate.IsZero():
		step.State = StateDone
		step.Done = f.PaidDate
	case asOf.After(due):
		step.State = StateMissed
		c.warn("Even the civil limitation period on this cheque expired on %s.", due.Pretty())
	case asOf.DaysUntil(due) <= 90:
		step.State = StateDue
	default:
		step.State = StatePending
	}
	c.add(step)
}

// --- stage and next action ------------------------------------------------

func stageOf(f Facts, p Periods, asOf, service Day, fatal bool) Stage {
	if !f.PaidDate.IsZero() {
		if !service.IsZero() && f.PaidDate.OnOrBefore(PaymentDeadline(service, p)) {
			return StagePaid
		}
		return StageSettledLate
	}
	if !f.ComplaintFiledDate.IsZero() {
		return StageFiled
	}
	if fatal {
		return StageBarred
	}
	switch {
	case service.IsZero():
		return StageNotice
	case asOf.After(PaymentDeadline(service, p)):
		return StageComplaint
	default:
		return StageWaiting
	}
}

// setNext picks the one thing the user should look at, and the date it is due.
//
// Only the four rungs of the section 138 cascade are candidates. The civil
// suit is a parallel remedy with years to run, not the next thing to do, and
// surfacing it as the next action would bury the fact that the criminal clock
// has stopped.
func (c *Cascade) setNext() {
	for _, key := range []string{"present", "notice", "pay", "complaint"} {
		s, ok := c.Step(key)
		if !ok {
			continue
		}
		switch s.State {
		case StatePending, StateDue, StateWaiting:
			c.NextKey, c.NextAction, c.NextDue, c.NextDaysLeft = s.Key, s.Action, s.Due, s.DaysLeft
			if s.Key == "complaint" && !c.FileBy.IsZero() {
				c.NextDue = c.FileBy
				c.NextDaysLeft = c.AsOf.DaysUntil(c.FileBy)
			}
			if s.Key == "pay" {
				c.NextAction = fmt.Sprintf("Wait. The drawer has until %s to pay. Diarise %s now: that is the date to file the complaint by.",
					s.Due.Pretty(), c.FileBy.Pretty())
			}
			return
		}
	}
	switch c.Stage {
	case StagePaid:
		c.NextAction = "Paid within the notice period. No offence arises; close the file."
	case StageFiled:
		c.NextAction = "Complaint filed. Nothing further on the limitation clock — the case now runs on the court's dates."
	case StageBarred:
		c.NextAction = "A statutory window has closed. Read the claim-ending problems below and take advice today."
	default:
		c.NextAction = "Nothing is due on the limitation clock."
	}
}
