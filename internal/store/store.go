// Package store manages the working directory: analysis cache, benchmarks,
// coaching sessions and the coach memory.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"

	"replayer/internal/analysis"
	"replayer/internal/replay"
)

// Store is the working directory. Aliases (alias -> main name) are applied
// to the analyses.
type Store struct {
	Dir     string
	Aliases map[string]string
}

func (s Store) path(parts ...string) string {
	return filepath.Join(append([]string{s.Dir}, parts...)...)
}

// MemoryDir holds the notes the coach keeps between sessions.
func (s Store) MemoryDir() (string, error) { return mkdir(s.path("memory")) }

// SessionDir is the working directory of a coaching session.
func (s Store) SessionDir(name string) (string, error) { return mkdir(s.path("sessions", name)) }

// BenchmarkDir holds the downloaded benchmark of a playlist and rank (not
// created: it is only read until a benchmark is built).
func (s Store) BenchmarkDir(playlist, rank string) string {
	return s.path("benchmarks", playlist+"_"+rank)
}

// Cached returns the cached analysis of r, nil if missing or outdated.
func (s Store) Cached(r *replay.Replay) *analysis.Report {
	b, err := os.ReadFile(s.path("analyses", r.ID+".json"))
	if err != nil {
		return nil
	}
	var rep analysis.Report
	if json.Unmarshal(b, &rep) != nil || rep.Version != analysis.Version || rep.AliasesKey != analysis.AliasesKey(s.Aliases) {
		return nil
	}
	return &rep
}

// Analysis returns the analysis of r from the cache, or computes and caches it.
func (s Store) Analysis(r *replay.Replay) (*analysis.Report, error) {
	if rep := s.Cached(r); rep != nil {
		return rep, nil
	}
	rep, err := analysis.File(r, s.Aliases)
	if err != nil {
		return nil, err
	}
	dir, err := mkdir(s.path("analyses"))
	if err != nil {
		return nil, err
	}
	return rep, WriteJSON(filepath.Join(dir, r.ID+".json"), rep)
}

func mkdir(dir string) (string, error) { return dir, os.MkdirAll(dir, 0o755) }

func WriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
