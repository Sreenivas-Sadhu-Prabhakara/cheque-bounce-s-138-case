// Command cheque-bounce-s-138-case keeps the statutory deadline cascade that
// follows a dishonoured cheque under section 138 of the Negotiable Instruments
// Act, 1881, on your own machine.
//
// It serves a small web UI on 127.0.0.1 and keeps everything in one local JSON
// file. It makes no outbound connections of any kind, and it never sends,
// serves or files anything: it prepares documents for you to issue yourself,
// and reminders for you to import into your own calendar.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sreenivas-sadhu-prabhakara/cheque-bounce-s-138-case/internal/s138"
)

//go:embed web
var webFiles embed.FS

// maxBodyBytes caps a request body. The UI sends a few kilobytes; anything
// larger is a mistake, and reading it unbounded would be one too.
const maxBodyBytes = 1 << 20

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "address to listen on; keep it on 127.0.0.1 so the tool stays on this machine")
	data := flag.String("data", "./cheque-case-data.json", "path to the local data file")
	flag.Parse()

	st, err := openStore(*data)
	if err != nil {
		log.Fatalf("cheque case clock: %v", err)
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           newServer(st).routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("cheque-bounce (s.138) case clock")
	log.Printf("open http://%s/ in your browser", *addr)
	log.Printf("data file: %s (nothing leaves this machine)", *data)
	log.Printf("this tool never sends, serves or files anything")
	log.Printf("press Ctrl+C to stop")

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("cheque case clock: %v", err)
	}
}

// server wires the JSON API and the embedded UI over one store.
type server struct {
	store *store
	// today is injectable so the handlers can be tested on a fixed date.
	today func() s138.Day
}

func newServer(st *store) *server {
	return &server{
		store: st,
		today: func() s138.Day { return s138.DayOf(time.Now()) },
	}
}

func (s *server) routes() http.Handler {
	ui, err := fs.Sub(webFiles, "web")
	if err != nil {
		// The directory is embedded at build time; this cannot fail at runtime.
		panic(fmt.Sprintf("embedded UI missing: %v", err))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/state", s.handleState)
	mux.HandleFunc("POST /api/cases", s.handleCreateCase)
	mux.HandleFunc("PUT /api/cases/{id}", s.handleUpdateCase)
	mux.HandleFunc("DELETE /api/cases/{id}", s.handleDeleteCase)
	mux.HandleFunc("POST /api/settings", s.handleSettings)
	mux.HandleFunc("POST /api/documents", s.handleDocument)
	mux.HandleFunc("GET /api/cases/{id}/calendar.ics", s.handleCalendar)
	mux.Handle("GET /", http.FileServerFS(ui))
	return headers(mux)
}

// headers adds the few response headers a local tool should still set.
func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// stateResponse is what the UI renders: the stored docket plus the whole
// cascade recomputed for every case on the chosen date.
type stateResponse struct {
	Office      s138.Office  `json:"office"`
	Periods     s138.Periods `json:"periods"`
	Statutory   s138.Periods `json:"statutoryPeriods"`
	PeriodLines []string     `json:"periodLines"`
	AsOf        s138.Day     `json:"asOf"`
	Today       s138.Day     `json:"today"`
	Cases       []caseView   `json:"cases"`
	Summary     summary      `json:"summary"`
	Disclaimer  string       `json:"disclaimer"`
}

// caseView is one stored case with its computed position.
type caseView struct {
	s138.Case
	Cascade s138.Cascade `json:"cascade"`
}

// summary is the one line the docket needs before anything else: how much of
// it is on fire.
type summary struct {
	Cases    int      `json:"cases"`
	Urgent   int      `json:"urgent"`
	Barred   int      `json:"barred"`
	Closed   int      `json:"closed"`
	NextRef  string   `json:"nextRef"`
	NextDue  s138.Day `json:"nextDue"`
	NextDays int      `json:"nextDays"`
	NextWhat string   `json:"nextWhat"`
}

func (s *server) state(asOf s138.Day) stateResponse {
	data := s.store.snapshot()
	periods := data.Periods.Normalise()

	views := make([]caseView, 0, len(data.Cases))
	sum := summary{Cases: len(data.Cases)}
	for _, c := range data.Cases {
		cascade := c.Cascade(periods, asOf)
		views = append(views, caseView{Case: c, Cascade: cascade})

		switch {
		case len(cascade.Fatal) > 0:
			sum.Barred++
		case !cascade.Alive:
			sum.Closed++
		}
		if !cascade.Alive || cascade.NextDue.IsZero() {
			continue
		}
		if cascade.NextDaysLeft <= s138.WarningDays {
			sum.Urgent++
		}
		// The nearest live deadline in the whole docket is what the user needs
		// at the top of the page.
		if sum.NextDue.IsZero() || cascade.NextDue.Before(sum.NextDue) {
			sum.NextRef, sum.NextDue = c.Ref, cascade.NextDue
			sum.NextDays, sum.NextWhat = cascade.NextDaysLeft, cascade.NextAction
		}
	}

	return stateResponse{
		Office:      data.Office,
		Periods:     periods,
		Statutory:   s138.DefaultPeriods(),
		PeriodLines: s138.PeriodLines(periods),
		AsOf:        asOf,
		Today:       s.today(),
		Cases:       views,
		Summary:     sum,
		Disclaimer:  s138.Disclaimer,
	}
}

func (s *server) handleState(w http.ResponseWriter, r *http.Request) {
	asOf, err := s.asOf(r.URL.Query().Get("asOf"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.state(asOf))
}

func (s *server) handleCreateCase(w http.ResponseWriter, r *http.Request) {
	in, err := s.readCase(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	in.ID = newID()

	if _, err := s.store.update(func(d *docket) error {
		d.Cases = append(d.Cases, in)
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.writeState(w, r, http.StatusCreated)
}

func (s *server) handleUpdateCase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	in, err := s.readCase(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	in.ID = id

	_, err = s.store.update(func(d *docket) error {
		for i := range d.Cases {
			if d.Cases[i].ID == id {
				d.Cases[i] = in
				return nil
			}
		}
		return errNotFound
	})
	switch {
	case errors.Is(err, errNotFound):
		writeError(w, http.StatusNotFound, fmt.Errorf("no case with that record key"))
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.writeState(w, r, http.StatusOK)
}

func (s *server) handleDeleteCase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := s.store.update(func(d *docket) error {
		for i := range d.Cases {
			if d.Cases[i].ID == id {
				d.Cases = append(d.Cases[:i], d.Cases[i+1:]...)
				return nil
			}
		}
		return errNotFound
	})
	switch {
	case errors.Is(err, errNotFound):
		writeError(w, http.StatusNotFound, fmt.Errorf("no case with that record key"))
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.writeState(w, r, http.StatusOK)
}

// readCase decodes, tidies and validates one case from the request body.
func (s *server) readCase(w http.ResponseWriter, r *http.Request) (s138.Case, error) {
	var in s138.Case
	if err := readJSON(w, r, &in); err != nil {
		return s138.Case{}, err
	}
	in = in.Tidy()
	if err := in.Validate(); err != nil {
		return s138.Case{}, err
	}
	return in, nil
}

// settingsRequest carries your own details and the periods to compute on.
type settingsRequest struct {
	Office  s138.Office  `json:"office"`
	Periods s138.Periods `json:"periods"`
}

func (s *server) handleSettings(w http.ResponseWriter, r *http.Request) {
	var in settingsRequest
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	in.Office = in.Office.Tidy()
	periods := in.Periods.Normalise()
	if err := periods.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if _, err := s.store.update(func(d *docket) error {
		d.Office = in.Office
		d.Periods = periods
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.writeState(w, r, http.StatusOK)
}

// documentRequest asks for one printable document. Nothing is sent or filed:
// the text comes back for the user to check, sign and issue themselves.
type documentRequest struct {
	Kind   string `json:"kind"`
	CaseID string `json:"caseId"`
	AsOf   string `json:"asOf"`
}

type documentResponse struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Text  string `json:"text"`
	AsOf  string `json:"asOf"`
}

func (s *server) handleDocument(w http.ResponseWriter, r *http.Request) {
	var in documentRequest
	if err := readJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	asOf, err := s.asOf(in.AsOf)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	data := s.store.snapshot()
	c, ok := find(data.Cases, in.CaseID)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("no case with that record key"))
		return
	}
	periods := data.Periods.Normalise()
	cascade := c.Cascade(periods, asOf)

	out := documentResponse{Kind: in.Kind, AsOf: asOf.String()}
	switch in.Kind {
	case "notice":
		out.Title = "Statutory demand notice — " + c.Title()
		out.Text = s138.DemandNotice(data.Office, c, cascade, asOf)
	case "schedule":
		out.Title = "Deadline schedule — " + c.Title()
		out.Text = s138.DeadlineSchedule(data.Office, c, cascade, asOf)
	case "particulars":
		out.Title = "Complaint particulars — " + c.Title()
		out.Text = s138.ComplaintParticulars(data.Office, c, cascade, asOf)
	case "checklist":
		out.Title = "Case file checklist — " + c.Title()
		out.Text = s138.AnnexureChecklist(data.Office, c, cascade, asOf)
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown document kind %q", in.Kind))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handleCalendar hands back an iCalendar file of the live deadlines for the
// user to import into their own calendar. This is the whole of the reminder
// feature: the tool cannot and does not send a message of any kind.
func (s *server) handleCalendar(w http.ResponseWriter, r *http.Request) {
	asOf, err := s.asOf(r.URL.Query().Get("asOf"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	data := s.store.snapshot()
	c, ok := find(data.Cases, r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Errorf("no case with that record key"))
		return
	}
	cascade := c.Cascade(data.Periods.Normalise(), asOf)

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", calendarFilename(c)))
	if _, err := w.Write([]byte(s138.CalendarICS(c, cascade, asOf))); err != nil {
		log.Printf("cheque case clock: writing calendar: %v", err)
	}
}

// calendarFilename keeps the download recognisable without letting a file
// reference out of the quoted header or onto the filesystem as a path.
func calendarFilename(c s138.Case) string {
	safe := make([]rune, 0, 40)
	for _, r := range c.Ref {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			safe = append(safe, r)
		default:
			safe = append(safe, '-')
		}
		if len(safe) == 40 {
			break
		}
	}
	name := strings.Trim(string(safe), "-")
	if name == "" {
		name = "cheque-case"
	}
	return name + "-deadlines.ics"
}

var errNotFound = errors.New("not found")

// asOf reads the date the cascade is computed for, defaulting to today.
func (s *server) asOf(raw string) (s138.Day, error) {
	day, err := s138.ParseDay(raw)
	if err != nil {
		return s138.Day{}, err
	}
	if day.IsZero() {
		return s.today(), nil
	}
	return day, nil
}

func (s *server) writeState(w http.ResponseWriter, r *http.Request, code int) {
	asOf, err := s.asOf(r.URL.Query().Get("asOf"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, code, s.state(asOf))
}

func find(cases []s138.Case, id string) (s138.Case, bool) {
	for _, c := range cases {
		if c.ID == id {
			return c, true
		}
	}
	return s138.Case{}, false
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("could not read the form: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("cheque case clock: writing response: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
