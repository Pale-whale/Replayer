package store

import (
	"os"
	"path/filepath"
	"testing"

	"replayer/internal/analysis"
	"replayer/internal/replay"
)

func TestCachedVersion(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	r := &replay.Replay{ID: "abc"}
	if s.Cached(r) != nil {
		t.Fatal("empty cache")
	}
	os.MkdirAll(filepath.Join(s.Dir, "analyses"), 0o755)
	WriteJSON(filepath.Join(s.Dir, "analyses", "abc.json"), analysis.Report{Version: analysis.Version - 1})
	if s.Cached(r) != nil {
		t.Error("outdated report must not be used")
	}
	WriteJSON(filepath.Join(s.Dir, "analyses", "abc.json"), analysis.Report{Version: analysis.Version, InPlaySeconds: 42})
	if rep := s.Cached(r); rep == nil || rep.InPlaySeconds != 42 {
		t.Errorf("cached report: %+v", rep)
	}
	s.Aliases = map[string]string{"Alt": "Main"}
	if s.Cached(r) != nil {
		t.Error("report made with other aliases must not be used")
	}
}
