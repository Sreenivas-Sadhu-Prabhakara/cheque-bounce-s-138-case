# Cheque bounce section 138 deadline tracker and demand notice generator

A single-binary tracker for the deadline cascade that follows a dishonoured cheque in **India**. Enter the date on the bank's return memo and it computes every downstream deadline under the Negotiable Instruments Act, 1881 — the thirty days to serve the demand notice, the drawer's fifteen days to pay, the one calendar month to file the complaint — and prints the notice, the schedule and the complaint particulars for you to issue yourself.

It runs on your own machine, serves a small web UI on `127.0.0.1`, keeps everything in one local JSON file, and makes no outbound connections of any kind.

> **This is a tracker, not legal advice.** It computes and prints; it never sends, serves or files anything, and it has no way to. Confirm the current statutory periods, and the facts and dates of your own case, with an advocate before acting on any date it shows.
>
> **A wrong date here is the worst bug this tool could have.** Every window below is either a condition of the offence or a period of limitation. Missing any one of them can extinguish the complaint, however good the rest of the case is — and unlike a crash, a wrong date looks exactly like a right one. That is why the arithmetic lives in `internal/s138` as pure functions, why every period is a named constant with its section reference, and why the tests exercise the day before, the day of and the day after every single window, in leap years and out of them.

## What it does

Enter one cheque at a time: the cheque and its date and amount, the drawer and the payee, the drawee bank, the bank it was presented through, and the date on the return memo. Only the memo date is needed to start the clock. The tool then shows:

- **the whole cascade** per case — the last day the cheque was presentable, the last day to serve the demand notice, the day the drawer's fifteen days expire, the day the cause of action arises, the last day to file the complaint, and the civil limitation date — each with the provision it comes from and the arithmetic that produced it;
- **one date to work to**, where the law or the date of service admits two readings: the earlier deadline leads and the outer limit sits beside it;
- **printable documents** — the statutory demand notice under clause (b) of the proviso to section 138, the deadline schedule, a complaint particulars sheet for an advocate to draft from, and a case-file checklist;
- **reminders you keep yourself** — an iCalendar file of the live deadlines with alarms at fourteen, seven, three and one day, which you download and import into your own calendar;
- **the docket at a glance** — the nearest deadline anywhere, how many cases fall due within a week, how many have a window already closed.

It never sends an email, never sends a message and never files anything. It prepares documents; you issue them.

## Who it's for

Solo advocates and small litigation firms in India carrying cheque-bounce matters, and whoever keeps the diary. A section 138 docket is a set of rigid, overlapping clocks, and a spreadsheet cannot recompute them when a date of service turns out to be different from the one assumed. It is equally usable by a payee — a trader, a supplier, a lender — who wants to understand the clock before instructing an advocate.

It is not a case-management system and it does not draft pleadings.

## The regulation it implements

**The Negotiable Instruments Act, 1881 — section 138 and section 142; the Limitation Act, 1963; the General Clauses Act, 1897.**

Section 138 makes the dishonour of a cheque an offence, but only if the three conditions in the proviso are met, in order:

```
(a)  the cheque was presented to the bank within six months of the date
     on which it is drawn, or within the period of its validity,
     whichever is EARLIER;

(b)  the payee or holder in due course made a demand for payment in
     writing, to the drawer, within THIRTY DAYS of receiving the
     information from the bank that the cheque was returned unpaid;

(c)  the drawer failed to pay within FIFTEEN DAYS of receiving that
     notice.
```

The offence is complete on the day after those fifteen days expire unpaid. That day is the cause of action, and **section 142(1)(b)** gives one month from it to file.

**One month, not thirty days.** Section 3(35) of the General Clauses Act, 1897 makes it a month reckoned by the British calendar — 28, 29, 30 or 31 days. The difference decides cases: a cause of action arising on 1 February 2023 gives a deadline of 1 March, while thirty days computes 3 March, two days out of time. The day a period runs *from* is excluded (s.9, General Clauses Act), and the month is reckoned from the day after the fifteen days expire — the reckoning in *Econ Antri Ltd v. Rom Industries Ltd*, (2014) 11 SCC 769, a three-judge bench affirming *Saketh India Ltd v. India Securities Ltd*, (1999) 3 SCC 1.

The whole chain on one worked example:

```
cheque dated                          01 December 2023
last day it was payable               01 March 2024      (+3 months)
return memo dated                     22 December 2023
last day to serve the demand notice   21 January 2024    (+30 days)
notice served on the drawer           16 January 2024
drawer's 15 days expire               31 January 2024    (+15 days)
cause of action arises                01 February 2024   (the next day)
LAST DAY TO FILE THE COMPLAINT        29 February 2024   (+1 month, clamped)
outer limit on the better view        01 March 2024
civil suit on the cheque              22 December 2026   (+3 years)
```

Two further things the tool does because the law makes them matter:

**Service.** Clause (c) runs from the drawer's *receipt* of the notice, and until the acknowledgement comes back that date is a guess. If the drawer received it sooner than assumed, the complaint window closes sooner too — so the tool also computes the deadline on the assumption that the notice arrived the day it was posted, and tells you to work to that. A complaint filed by it is in time whatever the court later finds about service. Section 27 of the General Clauses Act raises a presumption of service on a properly addressed, prepaid and posted notice; the drawer may rebut it.

**What is left when a window has gone.** A missed section 138 window is not necessarily a lost debt, and the tool prints the alternatives rather than just saying "too late": a cheque still within its validity can be presented again and a fresh dishonour gives a fresh cause of action (*MSR Leathers v. S. Palaniappan*, (2013) 1 SCC 177); the civil suit on the cheque runs for three years; the proviso to section 142(1)(b) lets a court take cognizance after the period on sufficient cause being shown; and section 147 makes the offence compoundable.

### The figures used

| Figure | Value | Where it comes from |
| --- | --- | --- |
| Demand notice | 30 days | s.138 proviso (b), NI Act 1881 — from the receipt of the bank's information |
| Drawer's time to pay | 15 days | s.138 proviso (c) — from the receipt of the notice |
| Complaint | 1 **calendar month** | s.142(1)(b), with s.3(35) General Clauses Act 1897 |
| Cheque validity | 3 months | RBI directions effective 1 April 2012 |
| Presentation | 6 months | s.138 proviso (a) — whichever of the two expires first |
| Assumed days in the post | 7 days | **not statutory** — this tool's working assumption until service is proved |
| Civil suit on the cheque | 3 years | Limitation Act, 1963 |
| Sentence | up to 2 years, or a fine up to 2× the cheque, or both | s.138 |
| Interim compensation | up to 20% of the cheque | s.143A (inserted 2018) |
| Appeal deposit | at least 20% | s.148 (inserted 2018) |

Every one of these is a named constant in `internal/s138/periods.go`, carrying the provision it comes from, and **every one is overridable in the tool**. The notice period was fifteen days until the 2002 amendment; set it and the whole docket recomputes, and every printed document states the figures it used. **These periods can change.** Confirm the current figures, and your own facts, with an advocate.

## Install and run

Go 1.25 or later. Standard library only — nothing to install.

```sh
git clone https://github.com/sreenivas-sadhu-prabhakara/cheque-bounce-s-138-case.git
cd cheque-bounce-s-138-case

# run it straight away
go run .

# or build the binary once and keep it
go build -o cheque-case .
./cheque-case

# options
./cheque-case -addr 127.0.0.1:8080 -data ./cheque-case-data.json
```

Then open <http://127.0.0.1:8080/>. Press Ctrl+C to stop.

| Flag | Default | What it does |
| --- | --- | --- |
| `-addr` | `127.0.0.1:8080` | Address to listen on. Keep it on 127.0.0.1 so the tool stays on this machine. |
| `-data` | `./cheque-case-data.json` | Where your docket is kept. |

## Where your data is stored

In the one JSON file named by `-data`, on your own machine. It holds your own details, the periods you are computing on and your cases, written atomically (temporary file, then rename) so an interrupted save cannot truncate a docket. A file that exists but cannot be parsed is an error rather than something to overwrite: a litigation docket is not the tool's to discard. You can read and edit the file in any text editor, and backing it up means copying one file.

No account, no server, no telemetry, no analytics, no fonts or scripts from anywhere else. The tool makes **no outbound connections of any kind** — the only network traffic is your own browser talking to the binary on 127.0.0.1. The reminders are an iCalendar file you download and import yourself. A tool holding privileged client material has no business reaching out to anyone, and this one has no code that could.

## Tests

```sh
gofmt -l .
go vet ./...
go test ./... -count=1
go build -o cheque-case .
```

The statutory rules live in `internal/s138` as pure functions over plain values, with table-driven tests over:

- **the date arithmetic** — thirty days off every month end, fifteen days across 29 February, calendar-month addition clamped to the end of the target month (31 January + 1 month is 28 or 29 February, never 2 March), year addition clamped the same way (29 February + 1 year is 28 February), leap and non-leap and century years, negative and zero offsets, and a test that pins the difference between one calendar month and thirty days in both directions;
- **every window boundary** — the day before, the day of and the day after the presentation limit, the demand-notice deadline, the drawer's fifteen days and both the safe and outer complaint deadlines, for a window not yet acted on and for one acted on early, on time, a day late and long past;
- **the worked authorities** — the dates in *Saketh India* are a test case;
- **the safety rules** — that an unproved date of service produces the earliest possible deadline, that an impossible sequence of dates is refused rather than computed, that a half-filled periods block in a hand-edited data file normalises to the Act rather than to zero, and that the cascade is pure;
- **the documents** — that the demand notice contains the demand clause (b) requires, for exactly the cheque amount in figures and in words, and that every document carries the disclaimer and says that nothing has been sent;
- **the calendar file** — RFC 5545 line folding at 75 octets, text escaping, all-day event dates, and that a closed window never becomes a reminder;
- **the HTTP surface** — validation, persistence, recomputation on a chosen date, and the download headers.

## Scope and accuracy

This is a deadline tracker. The dates it shows are computed from what you typed into it, and whether an offence under section 138 is made out turns on facts no tracker can see: whether the cheque was for a legally enforceable debt, whether the notice was properly addressed and served, whether the drawer is a company and who was in charge of its business, which court has jurisdiction under section 142(2). Where a date is unrecorded the tool says so rather than guessing; where dates are recorded in an impossible order it refuses the entry.

The statutory periods are the figures in force at the time of writing and can change. **Confirm the current periods and your own facts with an advocate, and file with days to spare rather than on the last day.** The documents this tool prints are drafts for you to check, settle, sign and issue; nothing is sent, served or filed for you.

MIT licensed.
