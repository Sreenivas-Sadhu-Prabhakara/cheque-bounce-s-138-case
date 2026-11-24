package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sreenivas-sadhu-prabhakara/cheque-bounce-s-138-case/internal/s138"
)

// fixedToday is the date the handlers are tested on, so nothing depends on
// when the suite runs.
var fixedToday = s138.NewDay(2024, time.February, 1)

func newTestServer(t *testing.T) (http.Handler, *server, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cheque-case-data.json")
	st, err := openStore(path)
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	srv := newServer(st)
	srv.today = func() s138.Day { return fixedToday }
	return srv.routes(), srv, path
}

func do(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeState(t *testing.T, rec *httptest.ResponseRecorder) stateResponse {
	t.Helper()
	var out stateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decoding the state: %v\n%s", err, rec.Body.String())
	}
	return out
}

func errorOf(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decoding the error: %v\n%s", err, rec.Body.String())
	}
	return out["error"]
}

// goodCase is the request body for a clean case: notice served 16 January
// 2024, so the complaint falls due on 29 February 2024.
const goodCase = `{
  "ref": "NI/2024/07",
  "payeeName": "Kaveri Engineering Works",
  "drawerName": "Veda Fabricators Private Limited",
  "drawerAddress": "22 Industrial Suburb, Bengaluru 560022",
  "consideration": "goods supplied under invoice KEW/2023/118",
  "chequeNo": "004512",
  "amount": 250000,
  "draweeBank": "Shivamogga Co-operative Bank",
  "draweeBranch": "Yeshwanthpur",
  "collectingBank": "Union Bank of India",
  "collectingBranch": "Peenya",
  "memoReason": "Funds insufficient",
  "chequeDate": "2023-12-01",
  "presentedDate": "2023-12-20",
  "memoDate": "2023-12-22",
  "noticeSentDate": "2024-01-10",
  "noticeServedDate": "2024-01-16"
}`

func addGoodCase(t *testing.T, h http.Handler) (stateResponse, string) {
	t.Helper()
	rec := do(t, h, "POST", "/api/cases", goodCase)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/cases = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	state := decodeState(t, rec)
	if len(state.Cases) != 1 {
		t.Fatalf("the docket has %d cases, want 1", len(state.Cases))
	}
	return state, state.Cases[0].ID
}

func TestStateStartsEmptyOnTheStatutoryPeriods(t *testing.T) {
	h, _, _ := newTestServer(t)
	rec := do(t, h, "GET", "/api/state", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/state = %d, want 200", rec.Code)
	}
	state := decodeState(t, rec)

	if len(state.Cases) != 0 {
		t.Errorf("a new docket has %d cases, want 0", len(state.Cases))
	}
	if state.Periods != s138.DefaultPeriods() {
		t.Errorf("a new docket computes on %+v, want the statutory periods", state.Periods)
	}
	if got, want := state.Today.String(), "2024-02-01"; got != want {
		t.Errorf("today = %s, want %s", got, want)
	}
	if got, want := state.AsOf.String(), "2024-02-01"; got != want {
		t.Errorf("asOf defaults to %s, want today (%s)", got, want)
	}
	if len(state.PeriodLines) == 0 {
		t.Error("the state should carry the periods with their sources for the UI to show")
	}
	if !strings.Contains(state.Disclaimer, "not legal advice") {
		t.Error("the state should carry the disclaimer")
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("API Cache-Control = %q, want no-store", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
}

func TestCreateCaseComputesTheWholeCascade(t *testing.T) {
	h, _, _ := newTestServer(t)
	state, _ := addGoodCase(t, h)
	view := state.Cases[0]

	if view.ID == "" {
		t.Error("a stored case needs a record key")
	}
	if got, want := view.Cascade.FileBy.String(), "2024-02-29"; got != want {
		t.Errorf("date to file by = %s, want %s", got, want)
	}
	if got, want := view.Cascade.CauseOfAction.String(), "2024-02-01"; got != want {
		t.Errorf("cause of action = %s, want %s", got, want)
	}
	if got, want := view.Cascade.Stage, s138.StageComplaint; got != want {
		t.Errorf("stage = %q, want %q", got, want)
	}
	if len(view.Cascade.Steps) != 5 {
		t.Errorf("the cascade has %d steps, want 5", len(view.Cascade.Steps))
	}
	if got, want := state.Summary.NextDue.String(), "2024-02-29"; got != want {
		t.Errorf("the docket's nearest deadline = %s, want %s", got, want)
	}
	if got, want := state.Summary.NextRef, "NI/2024/07"; got != want {
		t.Errorf("the docket's nearest deadline is on %q, want %q", got, want)
	}
	if got, want := state.Summary.Cases, 1; got != want {
		t.Errorf("summary counts %d cases, want %d", got, want)
	}
}

func TestCreateCaseRejectsWhatCannotBeComputed(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"no memo date", `{"ref":"A","drawerName":"D","chequeNo":"1","amount":100}`, "return memo"},
		{"no reference", `{"drawerName":"D","chequeNo":"1","amount":100,"memoDate":"2024-01-05"}`, "file reference"},
		{"no amount", `{"ref":"A","drawerName":"D","chequeNo":"1","memoDate":"2024-01-05"}`, "amount of the cheque"},
		{
			"dates out of order",
			`{"ref":"A","drawerName":"D","chequeNo":"1","amount":100,"memoDate":"2024-01-05","noticeSentDate":"2024-01-01"}`,
			"check the dates",
		},
		{
			"an impossible date",
			`{"ref":"A","drawerName":"D","chequeNo":"1","amount":100,"memoDate":"2023-02-29"}`,
			"invalid date",
		},
		{"not JSON", `{`, "could not read the form"},
		{
			"a field this tool does not know",
			`{"ref":"A","drawerName":"D","chequeNo":"1","amount":100,"memoDate":"2024-01-05","interestRate":18}`,
			"could not read the form",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, _, _ := newTestServer(t)
			rec := do(t, h, "POST", "/api/cases", c.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("POST /api/cases = %d, want 400: %s", rec.Code, rec.Body.String())
			}
			if msg := errorOf(t, rec); !strings.Contains(msg, c.wantErr) {
				t.Errorf("error = %q, want it to mention %q", msg, c.wantErr)
			}
		})
	}
}

func TestUpdateCase(t *testing.T) {
	h, _, _ := newTestServer(t)
	_, id := addGoodCase(t, h)

	// The acknowledgement comes back showing service a day earlier, which
	// moves the complaint deadline a day earlier too.
	body := strings.Replace(goodCase, `"noticeServedDate": "2024-01-16"`, `"noticeServedDate": "2024-01-15"`, 1)
	rec := do(t, h, "PUT", "/api/cases/"+id, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	state := decodeState(t, rec)
	if len(state.Cases) != 1 {
		t.Fatalf("the docket has %d cases, want 1", len(state.Cases))
	}
	if got, want := state.Cases[0].NoticeServedDate.String(), "2024-01-15"; got != want {
		t.Errorf("service date = %s, want %s", got, want)
	}
	// 15 Jan + 15 days = 30 Jan; one month clamps to 29 February.
	if got, want := state.Cases[0].Cascade.FileBy.String(), "2024-02-29"; got != want {
		t.Errorf("date to file by = %s, want %s", got, want)
	}
	if got, want := state.Cases[0].ID, id; got != want {
		t.Errorf("the record key changed to %q, want %q", got, want)
	}

	rec = do(t, h, "PUT", "/api/cases/nosuchkey", goodCase)
	if rec.Code != http.StatusNotFound {
		t.Errorf("PUT to an unknown key = %d, want 404", rec.Code)
	}
}

func TestDeleteCase(t *testing.T) {
	h, _, _ := newTestServer(t)
	_, id := addGoodCase(t, h)

	rec := do(t, h, "DELETE", "/api/cases/"+id, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if state := decodeState(t, rec); len(state.Cases) != 0 {
		t.Errorf("the docket still has %d cases", len(state.Cases))
	}
	if rec := do(t, h, "DELETE", "/api/cases/"+id, ""); rec.Code != http.StatusNotFound {
		t.Errorf("deleting twice = %d, want 404", rec.Code)
	}
}

func TestSettingsChangeEveryDeadline(t *testing.T) {
	h, _, _ := newTestServer(t)
	addGoodCase(t, h)

	// The pre-2002 fifteen-day notice period, for a historic file.
	rec := do(t, h, "POST", "/api/settings", `{
	  "office": {"advocate": "  A. N. Rao  ", "enrolment": "KAR/1234/2009"},
	  "periods": {"noticeDays": 15, "paymentDays": 15, "complaintMonths": 1,
	    "chequeValidityMonths": 3, "statutoryPresentationMonths": 6,
	    "deemedServiceDays": 7, "civilSuitYears": 3}
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/settings = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	state := decodeState(t, rec)

	if got, want := state.Office.Advocate, "A. N. Rao"; got != want {
		t.Errorf("the office name was stored as %q, want %q (trimmed)", got, want)
	}
	if got, want := state.Periods.NoticeDays, 15; got != want {
		t.Errorf("notice period = %d, want %d", got, want)
	}
	step, ok := state.Cases[0].Cascade.Step("notice")
	if !ok {
		t.Fatal("the cascade lost its notice step")
	}
	// 22 December 2023 + 15 days.
	if got, want := step.Due.String(), "2024-01-06"; got != want {
		t.Errorf("notice deadline on the fifteen-day period = %s, want %s", got, want)
	}
	if state.Cases[0].Cascade.Statutory {
		t.Error("a cascade computed on changed periods must not claim to be statutory")
	}
	if !strings.Contains(strings.Join(state.PeriodLines, "\n"), "15 days") {
		t.Error("the stated periods must be the ones actually used")
	}
}

func TestSettingsRejectImpossiblePeriods(t *testing.T) {
	h, _, _ := newTestServer(t)
	rec := do(t, h, "POST", "/api/settings", `{"office": {}, "periods": {"complaintMonths": 999}}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/settings = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	if msg := errorOf(t, rec); !strings.Contains(msg, "complaint period") {
		t.Errorf("error = %q, want it to name the offending period", msg)
	}
}

func TestSettingsFillBlankPeriodsFromTheAct(t *testing.T) {
	h, _, _ := newTestServer(t)
	rec := do(t, h, "POST", "/api/settings", `{"office": {"firm": "Rao & Associates"}, "periods": {}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/settings = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got := decodeState(t, rec).Periods; got != s138.DefaultPeriods() {
		t.Errorf("blank periods became %+v, want the statutory figures", got)
	}
}

func TestDocumentsAreReturnedAsTextAndNothingIsSent(t *testing.T) {
	h, _, _ := newTestServer(t)
	_, id := addGoodCase(t, h)

	wants := map[string][]string{
		"notice":      {"clause (b) of the proviso to section 138", "FIFTEEN (15) DAYS"},
		"schedule":    {"SECTION 138 DEADLINE SCHEDULE", "29 February 2024"},
		"particulars": {"COMPLAINT PARTICULARS", "Econ Antri"},
		"checklist":   {"CASE FILE CHECKLIST", "ESSENTIAL"},
	}
	for kind, phrases := range wants {
		t.Run(kind, func(t *testing.T) {
			rec := do(t, h, "POST", "/api/documents", `{"kind":"`+kind+`","caseId":"`+id+`","asOf":""}`)
			if rec.Code != http.StatusOK {
				t.Fatalf("POST /api/documents = %d, want 200: %s", rec.Code, rec.Body.String())
			}
			var doc documentResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
				t.Fatalf("decoding the document: %v", err)
			}
			flat := strings.Join(strings.Fields(doc.Text), " ")
			for _, want := range phrases {
				if !strings.Contains(flat, strings.Join(strings.Fields(want), " ")) {
					t.Errorf("the %s does not contain %q", kind, want)
				}
			}
			if !strings.Contains(flat, "NOTHING HAS BEEN SENT") {
				t.Errorf("the %s must say that nothing has been sent", kind)
			}
			if doc.Title == "" {
				t.Errorf("the %s has no title", kind)
			}
			if got, want := doc.AsOf, "2024-02-01"; got != want {
				t.Errorf("the %s is dated %s, want %s", kind, got, want)
			}
		})
	}

	if rec := do(t, h, "POST", "/api/documents", `{"kind":"affidavit","caseId":"`+id+`","asOf":""}`); rec.Code != http.StatusBadRequest {
		t.Errorf("an unknown document kind = %d, want 400", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/documents", `{"kind":"notice","caseId":"nope","asOf":""}`); rec.Code != http.StatusNotFound {
		t.Errorf("a document for an unknown case = %d, want 404", rec.Code)
	}
}

func TestCalendarIsADownloadTheUserImportsThemselves(t *testing.T) {
	h, _, _ := newTestServer(t)
	_, id := addGoodCase(t, h)

	rec := do(t, h, "GET", "/api/cases/"+id+"/calendar.ics", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET calendar = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got, want := rec.Header().Get("Content-Type"), "text/calendar; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if got, want := rec.Header().Get("Content-Disposition"), `attachment; filename="NI-2024-07-deadlines.ics"`; got != want {
		t.Errorf("Content-Disposition = %q, want %q", got, want)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, "BEGIN:VCALENDAR\r\n") || !strings.HasSuffix(body, "END:VCALENDAR\r\n") {
		t.Error("the response is not an iCalendar file")
	}
	if !strings.Contains(body, "DTSTART;VALUE=DATE:20240229") {
		t.Error("the calendar should carry the complaint deadline of 29 February 2024")
	}
	if rec := do(t, h, "GET", "/api/cases/nope/calendar.ics", ""); rec.Code != http.StatusNotFound {
		t.Errorf("a calendar for an unknown case = %d, want 404", rec.Code)
	}
}

func TestAsOfRecomputesTheWholePosition(t *testing.T) {
	h, _, _ := newTestServer(t)
	addGoodCase(t, h)

	// The day after the outer complaint limit, the claim is gone.
	rec := do(t, h, "GET", "/api/state?asOf=2024-03-02", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/state?asOf = %d, want 200", rec.Code)
	}
	state := decodeState(t, rec)
	if got, want := state.AsOf.String(), "2024-03-02"; got != want {
		t.Errorf("asOf = %s, want %s", got, want)
	}
	view := state.Cases[0]
	if len(view.Cascade.Fatal) == 0 {
		t.Error("past the complaint window the cascade must report the claim as ended")
	}
	if view.Cascade.Alive {
		t.Error("past the complaint window the case is not alive")
	}
	if got, want := state.Summary.Barred, 1; got != want {
		t.Errorf("summary counts %d barred cases, want %d", got, want)
	}

	// Well before the deadline it is simply pending.
	state = decodeState(t, do(t, h, "GET", "/api/state?asOf=2024-02-05", ""))
	if len(state.Cases[0].Cascade.Fatal) != 0 {
		t.Errorf("on 5 February the case is live, got %v", state.Cases[0].Cascade.Fatal)
	}
	if got, want := state.Summary.Urgent, 0; got != want {
		t.Errorf("summary counts %d urgent, want %d", got, want)
	}

	// A week out, it counts as urgent.
	state = decodeState(t, do(t, h, "GET", "/api/state?asOf=2024-02-25", ""))
	if got, want := state.Summary.Urgent, 1; got != want {
		t.Errorf("summary counts %d urgent on 25 February, want %d", got, want)
	}

	if rec := do(t, h, "GET", "/api/state?asOf=not-a-date", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("an unreadable asOf = %d, want 400", rec.Code)
	}
}

func TestSummaryCountsPaidCasesAsClosed(t *testing.T) {
	h, _, _ := newTestServer(t)
	body := strings.Replace(goodCase, `"noticeServedDate": "2024-01-16"`,
		`"noticeServedDate": "2024-01-16", "paidDate": "2024-01-25"`, 1)
	rec := do(t, h, "POST", "/api/cases", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/cases = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	state := decodeState(t, rec)
	if got, want := state.Cases[0].Cascade.Stage, s138.StagePaid; got != want {
		t.Errorf("stage = %q, want %q", got, want)
	}
	if got, want := state.Summary.Closed, 1; got != want {
		t.Errorf("summary counts %d closed, want %d", got, want)
	}
	if !state.Summary.NextDue.IsZero() {
		t.Errorf("a paid case should put nothing on the clock, got %s", state.Summary.NextDue)
	}
}

func TestDataFileIsWrittenAndReadBack(t *testing.T) {
	h, _, path := newTestServer(t)
	_, id := addGoodCase(t, h)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the data file was not written: %v", err)
	}
	if !strings.Contains(string(raw), "NI/2024/07") {
		t.Error("the data file does not contain the case")
	}

	// A second store over the same file must see the same docket.
	reopened, err := openStore(path)
	if err != nil {
		t.Fatalf("reopening the data file: %v", err)
	}
	data := reopened.snapshot()
	if len(data.Cases) != 1 || data.Cases[0].ID != id {
		t.Fatalf("the reopened docket is %+v, want the one case back", data.Cases)
	}
	if got, want := data.Cases[0].MemoDate.String(), "2023-12-22"; got != want {
		t.Errorf("the reopened memo date = %s, want %s", got, want)
	}
	if data.Periods != s138.DefaultPeriods() {
		t.Errorf("the reopened periods = %+v, want the statutory figures", data.Periods)
	}
}

func TestOpenStoreRefusesToOverwriteSomethingItDidNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notours.json")
	if err := os.WriteFile(path, []byte("this is somebody's notes, not a data file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openStore(path); err == nil {
		t.Error("openStore should refuse a file it cannot parse rather than overwrite it")
	}
}

func TestOpenStoreNormalisesAHandEditedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "edited.json")
	// A file with the periods block half filled in, as a hand edit leaves it.
	if err := os.WriteFile(path, []byte(`{"version":1,"periods":{"noticeDays":30},"cases":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := openStore(path)
	if err != nil {
		t.Fatalf("openStore: %v", err)
	}
	if got := st.snapshot().Periods; got != s138.DefaultPeriods() {
		t.Errorf("a half-filled periods block became %+v, want the statutory figures — a zero would wipe out a window", got)
	}
}

func TestStoreUpdateLeavesTheFileAloneWhenTheChangeFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	st, err := openStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.update(func(d *docket) error { d.Office.Firm = "First"; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := st.update(func(d *docket) error {
		d.Office.Firm = "Second"
		return errNotFound
	}); err == nil {
		t.Fatal("update should return the error the change reported")
	}
	if got, want := st.snapshot().Office.Firm, "First"; got != want {
		t.Errorf("a failed change left the docket saying %q, want %q", got, want)
	}
}

func TestEmbeddedUIIsServed(t *testing.T) {
	h, _, _ := newTestServer(t)
	for _, path := range []string{"/", "/assets/app.js", "/assets/style.css", "/assets/accent.css", "/assets/theme-init.js", "/assets/favicon.svg"} {
		rec := do(t, h, "GET", path, "")
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
		if rec.Body.Len() == 0 {
			t.Errorf("GET %s served nothing", path)
		}
	}
	// The file server canonicalises /index.html to /.
	if rec := do(t, h, "GET", "/index.html", ""); rec.Code != http.StatusMovedPermanently {
		t.Errorf("GET /index.html = %d, want a redirect to /", rec.Code)
	}
	if rec := do(t, h, "GET", "/", ""); !strings.Contains(rec.Body.String(), "Content-Security-Policy") {
		t.Error("the served page should carry its CSP")
	}
}

func TestCalendarFilenameIsSafe(t *testing.T) {
	cases := []struct{ ref, want string }{
		{"NI/2024/07", "NI-2024-07-deadlines.ics"},
		{"NI 2024 07", "NI-2024-07-deadlines.ics"},
		{"", "cheque-case-deadlines.ics"},
		{`../../etc/passwd`, "etc-passwd-deadlines.ics"},
		{`a"b`, "a-b-deadlines.ics"},
		{strings.Repeat("x", 80), strings.Repeat("x", 40) + "-deadlines.ics"},
	}
	for _, c := range cases {
		if got := calendarFilename(s138.Case{Ref: c.ref}); got != c.want {
			t.Errorf("calendarFilename(%q) = %q, want %q", c.ref, got, c.want)
		}
	}
}

func TestBodiesAreCapped(t *testing.T) {
	h, _, _ := newTestServer(t)
	huge := `{"ref":"` + strings.Repeat("x", 2<<20) + `"}`
	if rec := do(t, h, "POST", "/api/cases", huge); rec.Code != http.StatusBadRequest {
		t.Errorf("an oversized body = %d, want 400", rec.Code)
	}
}
