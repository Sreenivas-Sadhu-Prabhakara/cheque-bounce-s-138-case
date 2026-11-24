package s138

import (
	"fmt"
	"strings"
)

// Disclaimer goes at the foot of every document this tool prints. It is not
// boilerplate to be trimmed: a tracker that prints a statutory notice has to
// say what it is not.
const Disclaimer = "This document was prepared by a deadline tracker from the dates and particulars entered into it. " +
	"It is not legal advice and it is not a substitute for an advocate. " +
	"The statutory periods it works to are set out above with the sections they come from — confirm those periods, and the facts and dates of your own case, with an advocate before acting. " +
	"Nothing has been sent, served or filed: this is text for you to check, settle, sign and issue yourself."

// NotSentNotice is the line the tool puts above anything it hands back.
const NotSentNotice = "NOTHING HAS BEEN SENT. This tool has no way to send, serve or file anything. " +
	"It prints; you issue."

// doc builds a plain-text document line by line.
type doc struct{ b strings.Builder }

// docWidth is the column the documents fold at, so they print on A4 and read
// in a terminal without reflowing.
const docWidth = 72

// kvLabelWidth is the column the label-and-value lines line up on. Every label
// in these documents fits inside it, so the values all start in the same place
// and a long value folds under itself rather than off the page.
const kvLabelWidth = 30

func (d *doc) line(s string)                { d.b.WriteString(s); d.b.WriteByte('\n') }
func (d *doc) f(format string, args ...any) { d.line(fmt.Sprintf(format, args...)) }
func (d *doc) blank()                       { d.b.WriteByte('\n') }
func (d *doc) rule()                        { d.line(strings.Repeat("-", docWidth)) }
func (d *doc) String() string               { return d.b.String() }

// kv writes a label and its value, folding a long value under the label
// column. A value that will not fit in that column at all — a long file
// reference with nothing to break on — gets the label to itself and the full
// width of the page underneath, rather than running off the edge.
func (d *doc) kv(label string, value string) {
	label += ":"
	folded := hang(fmt.Sprintf("%-*s ", kvLabelWidth, label), value, docWidth)
	if longestLineOf(folded) <= docWidth {
		d.line(folded)
		return
	}
	d.line(label)
	d.wrapped("    ", value)
}

// longestLineOf measures the longest line of an already-folded block.
func longestLineOf(block string) int {
	longest := 0
	for _, l := range strings.Split(block, "\n") {
		if n := len([]rune(l)); n > longest {
			longest = n
		}
	}
	return longest
}

func (d *doc) kvd(label string, value Day) { d.kv(label, value.Pretty()) }

// wrapped writes text folded to the page width, with prefix on the first line
// and a matching run of spaces under it.
func (d *doc) wrapped(prefix string, text string) {
	if out := hang(prefix, text, docWidth); out != "" {
		d.line(out)
	}
}

// numbered writes a numbered paragraph hanging under its own number.
func (d *doc) numbered(n int, text string) {
	d.wrapped(fmt.Sprintf("%-4s", fmt.Sprintf("%d.", n)), text)
}

// wrap folds text to width columns on word boundaries. A single word longer
// than the width is left whole rather than cut: a cheque number or a case
// citation broken in half is worse than a long line.
func wrap(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		if len([]rune(line))+1+len([]rune(w)) > width {
			lines = append(lines, line)
			line = w
			continue
		}
		line += " " + w
	}
	return strings.Join(append(lines, line), "\n")
}

// hang folds text to width columns with prefix on the first line and an
// equal-width run of spaces on every line after it. A bullet or a number must
// never be repeated in the middle of a sentence, which is what prefixing every
// folded line would do.
func hang(prefix, text string, width int) string {
	body := wrap(text, width-len([]rune(prefix)))
	if body == "" {
		return ""
	}
	lines := strings.Split(body, "\n")
	continuation := strings.Repeat(" ", len([]rune(prefix)))
	for i := range lines {
		if i == 0 {
			lines[i] = prefix + lines[i]
			continue
		}
		lines[i] = continuation + lines[i]
	}
	return strings.Join(lines, "\n")
}

func (d *doc) footer() {
	d.blank()
	d.rule()
	d.wrapped("", NotSentNotice)
	d.blank()
	d.wrapped("", Disclaimer)
}

func (d *doc) heading(title string) {
	d.rule()
	d.line(strings.ToUpper(title))
	d.rule()
}

// signature block for the office issuing the document.
func (d *doc) signedBy(o Office) {
	d.blank()
	d.line("Yours faithfully,")
	d.blank()
	d.blank()
	d.line(Or(o.Advocate, "[ your name ]"))
	if o.Enrolment != "" {
		d.f("Advocate, enrolment no. %s", o.Enrolment)
	} else {
		d.line("Advocate")
	}
	if o.Firm != "" {
		d.line(o.Firm)
	}
	if o.Address != "" {
		d.wrapped("", o.Address)
	}
	if contact := strings.Join(nonEmpty(o.Phone, o.Email), "  ·  "); contact != "" {
		d.wrapped("", contact)
	}
}

func nonEmpty(values ...string) []string {
	var out []string
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// The demand notice under clause (b) of the proviso to section 138
// ---------------------------------------------------------------------------

// DemandNotice returns the statutory notice of demand as printable text.
//
// This is the document the whole cascade turns on: clause (b) of the proviso
// requires a demand in writing, given within thirty days, for the amount of
// the cheque, and clause (c) gives the drawer fifteen days from receiving it.
// A notice that demands something other than the cheque amount, or that omits
// the demand, puts the complaint at risk however well the dates were kept.
func DemandNotice(o Office, c Case, cas Cascade, asOf Day) string {
	d := &doc{}
	amount := FormatINR(c.Amount)
	words := AmountInWords(c.Amount)
	payee := Or(c.PayeeName, "[ your client's full name ]")

	d.line("BY REGISTERED POST WITH ACKNOWLEDGEMENT DUE")
	d.line("AND BY SPEED POST")
	d.blank()
	if o.Firm != "" {
		d.line(o.Firm)
	}
	if o.Advocate != "" {
		enrol := ""
		if o.Enrolment != "" {
			enrol = fmt.Sprintf(" (enrolment no. %s)", o.Enrolment)
		}
		d.f("%s, Advocate%s", o.Advocate, enrol)
	}
	if o.Address != "" {
		d.wrapped("", o.Address)
	}
	if contact := strings.Join(nonEmpty(o.Phone, o.Email), "  ·  "); contact != "" {
		d.wrapped("", contact)
	}
	d.blank()
	ref := Or(c.Ref, "[ your file reference ]")
	if oneLine := fmt.Sprintf("Ref: %-30s Date: %s", ref, asOf.Pretty()); len([]rune(oneLine)) <= docWidth {
		d.line(oneLine)
	} else {
		d.wrapped("Ref:  ", ref)
		d.f("Date: %s", asOf.Pretty())
	}
	d.blank()
	d.line("To,")
	d.f("    %s", c.DrawerName)
	for _, l := range addressLines(c.DrawerAddress) {
		d.wrapped("    ", l)
	}
	d.blank()
	d.wrapped("", fmt.Sprintf(
		"Sub: Statutory notice of demand under clause (b) of the proviso to section 138 of the Negotiable Instruments Act, 1881 — dishonour of cheque no. %s dated %s for Rs. %s.",
		Or(c.ChequeNo, "[ cheque number ]"), c.ChequeDate.Pretty(), amount))
	d.blank()
	d.line("Sir / Madam,")
	d.blank()
	d.wrapped("", fmt.Sprintf(
		"Under instructions from and on behalf of my client, %s%s, I serve upon you the following notice:",
		payee, addressClause(c.PayeeAddress)))
	d.blank()

	n := 0
	para := func(text string) {
		n++
		d.numbered(n, text)
		d.blank()
	}

	para(fmt.Sprintf(
		"That you are liable to pay my client a sum of Rs. %s (%s) on account of %s.",
		amount, words, Or(c.Consideration, "[ state here exactly what the cheque was given for — the invoices, the loan, the goods supplied, the agreement and its date ]")))

	para(fmt.Sprintf(
		"That in discharge of the said liability you drew and delivered to my client cheque no. %s dated %s for Rs. %s drawn on %s%s.",
		Or(c.ChequeNo, "[ cheque number ]"), c.ChequeDate.Pretty(), amount,
		Or(c.DraweeBank, "[ the drawee bank ]"), branchClause(c.DraweeBranch)))

	if !c.PresentedDate.IsZero() {
		para(fmt.Sprintf(
			"That my client presented the said cheque for collection on %s through %s%s.",
			c.PresentedDate.Pretty(), Or(c.CollectingBank, "[ your client's bank ]"), branchClause(c.CollectingBranch)))
	} else {
		para(fmt.Sprintf(
			"That my client presented the said cheque for collection in the ordinary course through %s%s.",
			Or(c.CollectingBank, "[ your client's bank ]"), branchClause(c.CollectingBranch)))
	}

	received := c.MemoReceivedDate
	if received.IsZero() {
		received = c.MemoDate
	}
	para(fmt.Sprintf(
		"That the said cheque was returned unpaid by %s, and my client's banker informed my client of the return by its memo dated %s bearing the reason \"%s\", which information was received by my client on %s.",
		Or(c.DraweeBank, "the drawee bank"), c.MemoDate.Pretty(),
		Or(c.MemoReason, "[ the reason printed on the return memo, word for word ]"), received.Pretty()))

	para("That the said cheque was drawn by you on an account maintained by you with the said bank, for the discharge of a legally enforceable debt or other liability owed to my client, and the dishonour of the cheque has caused my client loss and inconvenience.")

	para(fmt.Sprintf(
		"That by this notice, given within thirty days of the receipt by my client of the said information from the bank as required by clause (b) of the proviso to section 138 of the Negotiable Instruments Act, 1881, my client calls upon you to pay to my client the said sum of Rs. %s (%s) WITHIN FIFTEEN (15) DAYS of the receipt of this notice.",
		amount, words))

	para(fmt.Sprintf(
		"That should you fail to pay the said sum within the said period of fifteen days from the receipt of this notice, the cause of action under clause (c) of the proviso to section 138 will arise, and my client will be constrained to institute a complaint against you under section 138 of the Negotiable Instruments Act, 1881, the offence being punishable with imprisonment for a term which may extend to %d years, or with fine which may extend to twice the amount of the cheque, or with both, besides proceedings for the recovery of the said amount with interest and costs — all of which will be entirely at your risk, cost and consequences.",
		PunishmentYears))

	para("That a copy of this notice is retained in my office for record and for further necessary action.")

	d.signedBy(o)
	d.blank()
	d.line("Copy to: my client, for record.")

	d.blank()
	d.rule()
	d.line("BEFORE YOU SEND THIS")
	d.rule()
	for _, l := range noticeChecklist(c, cas) {
		d.wrapped("  - ", strings.TrimSpace(l))
	}
	d.footer()
	return d.String()
}

// noticeChecklist is the short list of things that most often go wrong with a
// section 138 notice, in the order they go wrong.
func noticeChecklist(c Case, cas Cascade) []string {
	notice, _ := cas.Step("notice")
	out := []string{
		fmt.Sprintf("Post it by %s. That is the thirty days in clause (b) counted from the memo date, which is the earliest the window can close.", notice.Due.Pretty()),
		"Check that the amount demanded is exactly the amount of the cheque. A notice that demands more — interest, costs, other invoices — risks being read as not making the demand clause (b) requires. Claim the extras in the complaint and in the civil suit, not here.",
		"Check the drawer's address against the cheque, the bank's records and the latest correspondence. Send it to every address you have, and to the registered office as well if the drawer is a company.",
		"Where the drawer is a company, firm or LLP, name the company as the first addressee and name the signatories and the persons in charge separately — section 141 reaches them, but only if they are put on notice and arraigned.",
		"Post it by registered post with acknowledgement due AND by speed post, and keep the receipts, the tracking printouts and the acknowledgement card. Those receipts are the proof of service the whole case stands on.",
		"Attach nothing that contradicts the notice, and keep an office copy signed and dated.",
	}
	if c.MemoReason == "" {
		out = append(out, "Fill in the reason printed on the return memo, word for word. \"Funds insufficient\", \"exceeds arrangement\", \"payment stopped by drawer\" and \"account closed\" are all within section 138; a memo returned for a technical reason may not be.")
	}
	return out
}

func addressLines(address string) []string {
	address = strings.TrimSpace(address)
	if address == "" {
		return []string{"[ the drawer's full address, as on the cheque and the bank's records ]"}
	}
	var out []string
	for _, part := range strings.Split(strings.ReplaceAll(address, "\r\n", "\n"), "\n") {
		for _, piece := range strings.Split(part, ",") {
			if p := strings.TrimSpace(piece); p != "" {
				out = append(out, p+",")
			}
		}
	}
	if len(out) > 0 {
		out[len(out)-1] = strings.TrimSuffix(out[len(out)-1], ",")
	}
	return out
}

func addressClause(address string) string {
	if strings.TrimSpace(address) == "" {
		return ""
	}
	return ", of " + strings.Join(strings.Fields(strings.ReplaceAll(address, "\n", " ")), " ")
}

func branchClause(branch string) string {
	if strings.TrimSpace(branch) == "" {
		return ""
	}
	return ", " + branch + " branch"
}

// ---------------------------------------------------------------------------
// The deadline schedule
// ---------------------------------------------------------------------------

// DeadlineSchedule returns the whole cascade as a printable page: every
// milestone, the date it falls, the section it comes from and the arithmetic
// that produced it. This is the page to pin to the file.
func DeadlineSchedule(o Office, c Case, cas Cascade, asOf Day) string {
	d := &doc{}
	d.heading("Section 138 deadline schedule")
	d.kv("File reference", Or(c.Ref, "—"))
	d.kv("Drawer", Or(c.DrawerName, "—"))
	d.kv("Payee", Or(c.PayeeName, "—"))
	d.kv("Cheque", fmt.Sprintf("no. %s dated %s for Rs. %s",
		Or(c.ChequeNo, "—"), c.ChequeDate.Pretty(), FormatINR(c.Amount)))
	d.kv("Drawee bank", strings.TrimSpace(Or(c.DraweeBank, "—")+branchClause(c.DraweeBranch)))
	d.kvd("Dishonour memo dated", c.MemoDate)
	d.kv("Reason on the memo", Or(c.MemoReason, "—"))
	d.kv("Schedule computed as on", asOf.Pretty())
	d.kv("Stage", cas.StageLabel)
	d.blank()

	d.rule()
	d.line("NEXT")
	d.rule()
	if !cas.NextDue.IsZero() {
		d.f("%s  (%s)", cas.NextDue.Pretty(), daysPhrase(cas.NextDaysLeft))
	}
	d.wrapped("", cas.NextAction)
	d.blank()

	d.rule()
	d.line("THE CASCADE")
	d.rule()
	for _, s := range cas.Steps {
		d.f("[%s] %s", strings.ToUpper(string(s.State)), s.Name)
		d.f("     %s  ·  %s", s.Section, s.Actor)
		if !s.From.IsZero() {
			d.wrapped("     counted from  ", fmt.Sprintf("%s  (%s)", s.From.Pretty(), s.FromWhat))
		}
		d.wrapped("     rule          ", s.Rule)
		if !s.Due.IsZero() {
			weekend := ""
			if s.Weekend {
				weekend = fmt.Sprintf("  [falls on a %s]", s.Due.Weekday())
			}
			d.f("     last day      %s%s", s.Due.Pretty(), weekend)
		}
		if !s.Outer.IsZero() {
			d.f("     outer limit   %s", s.Outer.Pretty())
		}
		if !s.Done.IsZero() {
			d.f("     done on       %s", s.Done.Pretty())
		}
		if s.Note != "" {
			d.wrapped("     ", s.Note)
		}
		d.blank()
	}

	if !cas.FileBy.IsZero() {
		d.rule()
		d.line("THE ONE DATE TO REMEMBER")
		d.rule()
		d.f("File the complaint on or before  %s", cas.FileBy.Pretty())
		d.wrapped("", fmt.Sprintf("The cause of action arose, or will arise, on %s. The complaint window is one calendar month from the expiry of the drawer's %d days, and the date above is the earliest that window can close on the dates recorded. Filing by it is in time whatever the court later finds about when the notice was served.",
			cas.CauseOfAction.Pretty(), cas.Periods.PaymentDays))
		d.blank()
	}

	if len(cas.Fatal) > 0 {
		d.rule()
		d.line("CLAIM-ENDING PROBLEMS")
		d.rule()
		for i, f := range cas.Fatal {
			d.numbered(i+1, f)
			d.blank()
		}
	}
	if len(cas.Warnings) > 0 {
		d.rule()
		d.line("WARNINGS")
		d.rule()
		for i, w := range cas.Warnings {
			d.numbered(i+1, w)
			d.blank()
		}
	}

	d.rule()
	d.line("THE PERIODS THIS SCHEDULE WAS COMPUTED ON")
	d.rule()
	for _, l := range PeriodLines(cas.Periods) {
		d.wrapped("  ", l)
	}
	if !cas.Statutory {
		d.blank()
		d.wrapped("", "NOTE: one or more of these periods has been changed from the figure in the Act. Every date above was computed on the figures shown, not on the statutory ones.")
	}
	d.footer()
	return d.String()
}

// PeriodLines states the periods a computation ran on, each with its source.
// The UI, the printed schedule and the complaint particulars all use this one
// list, so none of them can drift from the constants.
func PeriodLines(p Periods) []string {
	p = p.Normalise()
	return []string{
		fmt.Sprintf("Demand notice: %d days  —  s.138 proviso (b), from the receipt of the bank's information", p.NoticeDays),
		fmt.Sprintf("Drawer's time to pay: %d days  —  s.138 proviso (c), from the receipt of the notice", p.PaymentDays),
		fmt.Sprintf("Complaint: %d calendar month  —  s.142(1)(b), from the date the cause of action arises; a calendar month under s.3(35) of the General Clauses Act, 1897, not 30 days", p.ComplaintMonths),
		fmt.Sprintf("Cheque validity: %d months  —  Reserve Bank of India directions effective 1 April 2012", p.ChequeValidityMonths),
		fmt.Sprintf("Presentation: %d months  —  s.138 proviso (a), whichever of the two expires first", p.StatutoryPresentationMonths),
		fmt.Sprintf("Assumed days in the post: %d  —  NOT a statutory period; this tool's working assumption until service is proved", p.DeemedServiceDays),
		fmt.Sprintf("Civil suit on the cheque: %d years  —  Limitation Act, 1963; confirm the article that applies", p.CivilSuitYears),
	}
}

func daysPhrase(days int) string {
	switch {
	case days < -1:
		return fmt.Sprintf("%d days ago", -days)
	case days == -1:
		return "yesterday"
	case days == 0:
		return "today"
	case days == 1:
		return "tomorrow"
	default:
		return fmt.Sprintf("in %d days", days)
	}
}

// ---------------------------------------------------------------------------
// Complaint particulars
// ---------------------------------------------------------------------------

// ComplaintParticulars returns the facts sheet an advocate drafts the
// complaint from: the parties, the cheque, the dates, the cause-of-action
// paragraph with its arithmetic spelt out, the jurisdiction limb relied on and
// the limitation statement.
//
// It is deliberately not a draft complaint. The pleading is the advocate's
// work; what a tracker can do is make sure not one date or particular is
// missing from it.
func ComplaintParticulars(o Office, c Case, cas Cascade, asOf Day) string {
	d := &doc{}
	d.heading("Complaint particulars — s.138 r/w s.142, NI Act 1881")
	d.wrapped("", "For your advocate to draft from. Every figure and date below comes from the case record; nothing here is a pleading.")
	d.blank()
	d.kv("File reference", Or(c.Ref, "—"))
	d.kv("Office", Or(strings.Join(nonEmpty(o.Firm, o.Advocate), " · "), "—"))
	d.kv("Particulars as on", asOf.Pretty())
	d.blank()

	d.line("1.  PARTIES")
	d.kv("    Complainant (payee)", Or(c.PayeeName, "[ ]"))
	d.kv("    Complainant's address", Or(oneLine(c.PayeeAddress), "[ ]"))
	d.kv("    Accused (drawer)", Or(c.DrawerName, "[ ]"))
	d.kv("    Accused's address", Or(oneLine(c.DrawerAddress), "[ ]"))
	d.wrapped("    ", "Section 142(1)(a) requires the complaint to be made in writing by the payee or the holder in due course. Where the accused is a company, firm or LLP, section 141 reaches the persons in charge of and responsible for the conduct of its business and the signatory, who must be named and arraigned separately, with the averment that they were in charge at the material time.")
	d.blank()

	d.line("2.  THE DEBT OR LIABILITY")
	d.wrapped("    ", Or(c.Consideration, "[ state the debt or other liability the cheque was given to discharge, with the invoices, agreement or account it arises from ]"))
	d.blank()

	d.line("3.  THE CHEQUE")
	d.kv("    Number", Or(c.ChequeNo, "[ ]"))
	d.kvd("    Date on the cheque", c.ChequeDate)
	d.kv("    Amount", fmt.Sprintf("Rs. %s  (%s)", FormatINR(c.Amount), AmountInWords(c.Amount)))
	d.kv("    Drawn on", strings.TrimSpace(Or(c.DraweeBank, "[ ]")+branchClause(c.DraweeBranch)))
	d.kv("    Presented through", strings.TrimSpace(Or(c.CollectingBank, "[ ]")+branchClause(c.CollectingBranch)))
	d.kvd("    Presented on", c.PresentedDate)
	d.blank()

	d.line("4.  THE DISHONOUR")
	d.kvd("    Memo dated", c.MemoDate)
	d.kv("    Reason on the memo", Or(c.MemoReason, "[ word for word from the memo ]"))
	d.kvd("    Information received on", Later(c.MemoReceivedDate, c.MemoDate))
	d.blank()

	d.line("5.  THE DEMAND NOTICE")
	d.kvd("    Despatched on", c.NoticeSentDate)
	d.kv("    Mode", Or(c.NoticeMode, "[ registered post with A/D, speed post, courier, email ]"))
	d.kvd("    Received by the drawer on", c.NoticeServedDate)
	if cas.ServiceBasis == "deemed" {
		d.wrapped("    ", fmt.Sprintf("Service is not yet proved. The dates below assume receipt on %s, being despatch plus %d days in the post — this tool's assumption, not a statutory period. Section 27 of the General Clauses Act, 1897 raises a presumption of service where the notice was properly addressed, prepaid and posted; the drawer may rebut it.", cas.ServiceDate.Pretty(), cas.Periods.DeemedServiceDays))
	}
	d.blank()

	d.line("6.  CAUSE OF ACTION AND LIMITATION")
	if cas.CauseOfAction.IsZero() {
		d.wrapped("    ", "The cause of action has not arisen yet: the demand notice has not gone out, or its service date is not known.")
	} else {
		pay, _ := cas.Step("pay")
		complaint, _ := cas.Step("complaint")
		d.f("    Notice received (or deemed received)       %s", cas.ServiceDate.Pretty())
		d.f("    + %-2d days under proviso (c), expiring     %s", cas.Periods.PaymentDays, pay.Due.Pretty())
		d.f("    Cause of action arose on                   %s", cas.CauseOfAction.Pretty())
		d.f("    + %d calendar month under s.142(1)(b)       %s", cas.Periods.ComplaintMonths, complaint.Due.Pretty())
		if !complaint.Outer.IsZero() {
			d.f("    Outer limit on the better view             %s", complaint.Outer.Pretty())
		}
		d.f("    DATE TO FILE BY                            %s", cas.FileBy.Pretty())
		if !c.ComplaintFiledDate.IsZero() {
			d.f("    Filed on                                   %s", c.ComplaintFiledDate.Pretty())
		}
		d.blank()
		d.wrapped("    ", "The day the notice was received is excluded from the fifteen days, and the offence is complete only on the day after they expire unpaid. The month under section 142(1)(b) is a calendar month reckoned from the cause of action, excluding that day: Econ Antri Ltd v. Rom Industries Ltd, (2014) 11 SCC 769, a three-judge bench affirming Saketh India Ltd v. India Securities Ltd, (1999) 3 SCC 1. Where the two readings differ by a day, the date to file by above is the earlier.")
	}
	d.blank()

	d.line("7.  JURISDICTION")
	d.kv("    Court", Or(c.CourtName, "[ the Metropolitan Magistrate or JMFC with jurisdiction ]"))
	d.kv("    Case number", Or(c.CaseNo, "[ once numbered ]"))
	d.wrapped("    ", "Section 142(2)(a): where the cheque is delivered for collection through an account, the offence is inquired into and tried by the court within whose local jurisdiction the branch of the bank where the payee maintains that account is situated. Section 142(2)(b) applies where the cheque was presented otherwise than through an account. Section 142(1)(c): no court inferior to that of a Metropolitan Magistrate or a Judicial Magistrate of the first class shall try the offence.")
	d.blank()

	d.line("8.  RELIEF AND WHAT TO ASK FOR")
	d.wrapped("    ", fmt.Sprintf("Section 138: imprisonment up to %d years, or fine up to twice the amount of the cheque, or both. Section 143A: interim compensation of up to %d%% of the cheque amount — Rs. %s on this cheque — which the trial court may direct the drawer to pay. Section 148: on an appeal against conviction, a deposit of a minimum of %d%% of the fine or compensation. Section 147: the offence is compoundable.",
		PunishmentYears, InterimCompensationMaxPct, FormatINR(c.Amount*InterimCompensationMaxPct/100), AppealDepositMinPct))
	d.blank()

	d.line("9.  EVIDENCE AND PROCEDURE")
	d.wrapped("    ", "Section 145(1): the complainant's evidence may be given on affidavit, and read in evidence in any inquiry, trial or other proceeding, subject to section 145(2). Section 146: the bank's slip or memo bearing the official mark denoting dishonour raises a presumption of dishonour unless the contrary is proved. Section 139: unless the contrary is proved, it is presumed that the holder received the cheque for the discharge of a debt or other liability.")
	d.blank()

	d.line("10. ANNEXURES")
	for _, item := range annexures(c) {
		d.wrapped("    - ", item)
	}

	if !cas.FileBy.IsZero() {
		d.blank()
		d.rule()
		d.f("FILE BY %s", strings.ToUpper(cas.FileBy.Pretty()))
		d.rule()
	}
	d.footer()
	return d.String()
}

// AnnexureChecklist is the list of documents to put on the file, on its own.
func AnnexureChecklist(o Office, c Case, cas Cascade, asOf Day) string {
	d := &doc{}
	d.heading("Case file checklist")
	d.kv("Office", Or(strings.Join(nonEmpty(o.Firm, o.Advocate), " · "), "—"))
	d.kv("File reference", Or(c.Ref, "—"))
	d.kv("Drawer", Or(c.DrawerName, "—"))
	d.kv("Cheque", fmt.Sprintf("no. %s for Rs. %s", Or(c.ChequeNo, "—"), FormatINR(c.Amount)))
	d.kv("As on", asOf.Pretty())
	d.blank()
	d.wrapped("", "Tick each one as it goes on the file. The items marked ESSENTIAL are the ones a complaint under section 138 is routinely dismissed for want of.")
	d.blank()
	for i, item := range annexures(c) {
		d.numbered(i+1, "[ ] "+item)
	}
	d.blank()
	d.rule()
	d.line("IF A WINDOW HAS BEEN MISSED")
	d.rule()
	for _, item := range recoveryOptions(c, cas) {
		d.wrapped("  - ", item)
	}
	d.footer()
	return d.String()
}

func annexures(c Case) []string {
	items := []string{
		"ESSENTIAL — the original cheque no. " + Or(c.ChequeNo, "[ ]") + ", and a legible copy for the record.",
		"ESSENTIAL — the original bank return or dishonour memo dated " + c.MemoDate.Pretty() + ", bearing the bank's mark and the reason for return.",
		"ESSENTIAL — the office copy of the demand notice, signed and dated " + c.NoticeSentDate.Pretty() + ".",
		"ESSENTIAL — the postal receipts for the registered post and the speed post, with the article numbers legible.",
		"ESSENTIAL — the acknowledgement card, the delivery or tracking record, or the returned unclaimed envelope with the postal endorsement intact. Do not open a returned envelope.",
		"ESSENTIAL — the complainant's affidavit under section 145(1) of the Negotiable Instruments Act, 1881.",
		"The documents that establish the debt the cheque was given for: invoices, the ledger or statement of account, the agreement, the delivery records, the loan papers.",
		"The complainant's bank statement or pass book showing the cheque being presented and returned.",
		"Where the complainant is a company, firm or LLP: the board resolution or authorisation naming the person competent to complain, and proof of their authority.",
		"Where the accused is a company, firm or LLP: the master data or registration particulars showing the registered office and the directors or partners, as on the date of the cheque.",
		"Vakalatnama, and the list of dates and events.",
		"A copy of this deadline schedule, so the limitation position on the file is on the file.",
	}
	if c.NoticeMode != "" {
		items = append(items, "Proof of the additional mode used ("+c.NoticeMode+"): the courier's delivery record, or the sent copy and delivery report of the email.")
	}
	return items
}

// recoveryOptions is what is left when a window has closed. A tracker that
// only ever says "too late" is of no use on the day it matters most.
func recoveryOptions(c Case, cas Cascade) []string {
	out := []string{
		"Take advice today. Everything below depends on facts a tracker cannot see.",
	}
	if !c.ChequeDate.IsZero() {
		effective, _ := LastPresentationDate(c.ChequeDate, cas.Periods)
		out = append(out, fmt.Sprintf(
			"Present the cheque again, if it is still within its validity — that is, on or before %s. A fresh dishonour gives a fresh cause of action and a fresh thirty days for the notice, and a payee who did not act on an earlier dishonour is not barred from acting on a later one: MSR Leathers v. S. Palaniappan, (2013) 1 SCC 177.",
			effective.Pretty()))
	} else {
		out = append(out, "Present the cheque again, if it is still within its validity. A fresh dishonour gives a fresh cause of action: MSR Leathers v. S. Palaniappan, (2013) 1 SCC 177.")
	}
	out = append(out, fmt.Sprintf(
		"The civil track is still open until %s: a suit for the money, usually a summary suit under Order XXXVII of the Code of Civil Procedure, 1908, where the cheque itself is the written instrument sued on.",
		CivilSuitDeadline(c.MemoDate, cas.Periods).Pretty()))
	out = append(out,
		"Where only the complaint is out of time, the proviso to section 142(1)(b) lets the court take cognizance after the period on being satisfied that there was sufficient cause for not filing in time. Prepare the application and the material that shows the cause; do not assume it.",
		"Section 147 makes the offence compoundable, so a negotiated settlement is open at any stage and is often the faster road to the money.")
	return out
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\n", " ")), " ")
}
