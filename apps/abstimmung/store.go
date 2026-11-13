package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"sync"
)

// maxStimmen bounds the file: a guest who throws their cookie away is a new
// voter each time, and the page is on the internet.
const maxStimmen = 5000

var errVoll = errors.New("the poll is full")

// Speicher is every vote, by voter id, in memory and in one JSON file. A
// write goes to a temp file beside it and is renamed over it, so a crash
// leaves the old file or the new one, never half of one.
type Speicher struct {
	mu      sync.Mutex
	pfad    string
	stimmen map[string]Stimme
}

type datei struct {
	Stimmen map[string]Stimme `json:"stimmen"`
}

// LadeSpeicher reads the file, or starts empty when there is none. A file
// that cannot be read is an error, never an empty poll written over it;
// so is a folder the program cannot write (the unit's UserNS line).
func LadeSpeicher(pfad string) (*Speicher, error) {
	s := &Speicher{pfad: pfad, stimmen: map[string]Stimme{}}
	raw, err := os.ReadFile(pfad)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		var d datei
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, fmt.Errorf("%s is not readable as votes: %w", pfad, err)
		}
		if d.Stimmen != nil {
			s.stimmen = d.Stimmen
		}
	}
	probe, err := os.CreateTemp(filepath.Dir(pfad), ".probe-*")
	if err != nil {
		return nil, fmt.Errorf("the data folder is not writable: %w", err)
	}
	probe.Close()
	return s, os.Remove(probe.Name())
}

// Setze stores id's vote, replacing an earlier one; memory changes only
// after the file did.
func (s *Speicher) Setze(id string, st Stimme) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, da := s.stimmen[id]; !da && len(s.stimmen) >= maxStimmen {
		return errVoll
	}
	neu := maps.Clone(s.stimmen)
	neu[id] = st
	if err := schreibe(s.pfad, datei{Stimmen: neu}); err != nil {
		return err
	}
	s.stimmen = neu
	return nil
}

func (s *Speicher) Hole(id string) (Stimme, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.stimmen[id]
	return st, ok
}

// Alle is every vote, without the voter ids.
func (s *Speicher) Alle() []Stimme {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Stimme, 0, len(s.stimmen))
	for _, st := range s.stimmen {
		out = append(out, st)
	}
	return out
}

func schreibe(pfad string, d datei) error {
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(pfad), ".stimmen-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // gone already after the rename
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), pfad)
}
