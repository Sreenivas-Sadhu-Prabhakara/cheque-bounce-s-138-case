package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sreenivas-sadhu-prabhakara/cheque-bounce-s-138-case/internal/s138"
)

// fileVersion is stamped into the data file so a later version of the tool can
// recognise what it is reading.
const fileVersion = 1

// docket is everything the tool keeps: your own details, the periods in force
// and the cases. It is the entire contents of the data file.
type docket struct {
	Version int          `json:"version"`
	Office  s138.Office  `json:"office"`
	Periods s138.Periods `json:"periods"`
	Cases   []s138.Case  `json:"cases"`
}

// store holds the docket in memory and writes it to one local JSON file.
// Nothing is sent anywhere: the file is the whole database.
type store struct {
	mu   sync.Mutex
	path string
	data docket
}

// openStore reads the data file, or starts an empty docket when there is none.
// A file that exists but cannot be parsed is an error rather than something to
// overwrite — a litigation docket is not ours to discard.
func openStore(path string) (*store, error) {
	s := &store{
		path: path,
		data: docket{Version: fileVersion, Periods: s138.DefaultPeriods()},
	}
	raw, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return s, nil
	case err != nil:
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var data docket
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("%s is not a data file this tool wrote: %w", path, err)
	}
	// A file written by an older version, or edited by hand, may be missing a
	// period. Normalise rather than compute a deadline off a zero.
	data.Periods = data.Periods.Normalise()
	data.Version = fileVersion
	s.data = data
	return s, nil
}

// snapshot returns a copy of the docket that the caller can read freely.
func (s *store) snapshot() docket {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.copyLocked()
}

// update applies fn to the docket and writes the file. The docket is only
// changed in memory if the write succeeds, so a failed write leaves the tool
// and the file saying the same thing.
func (s *store) update(fn func(*docket) error) (docket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.copyLocked()
	if err := fn(&next); err != nil {
		return docket{}, err
	}
	previous := s.data
	s.data = next
	if err := s.writeLocked(); err != nil {
		s.data = previous
		return docket{}, err
	}
	return s.copyLocked(), nil
}

func (s *store) copyLocked() docket {
	out := s.data
	out.Cases = make([]s138.Case, len(s.data.Cases))
	copy(out.Cases, s.data.Cases)
	return out
}

// writeLocked writes the docket to a temporary file in the same directory and
// renames it over the data file, so an interrupted write cannot truncate the
// user's records.
func (s *store) writeLocked() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".cheque-case-*.tmp")
	if err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	name := tmp.Name()
	defer os.Remove(name) // a no-op once the rename has succeeded

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", s.path, err)
	}
	if err := os.Rename(name, s.path); err != nil {
		return fmt.Errorf("replacing %s: %w", s.path, err)
	}
	return nil
}

// newID returns a short random key for a new case record. It identifies the
// row in the local file and in the UI; it is not part of the court record.
func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand does not fail in practice; fall back to something still
		// unique rather than handing out an empty key.
		return fmt.Sprintf("t%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
