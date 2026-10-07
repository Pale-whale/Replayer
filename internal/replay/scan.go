package replay

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Scan parses every .replay file in dir, newest first. Files that fail to
// parse are reported in errs and skipped.
func Scan(dir string) (replays []*Replay, errs []error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, []error{err}
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".replay") {
			continue
		}
		r, err := ParseFile(filepath.Join(dir, e.Name()))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		replays = append(replays, r)
	}
	sort.Slice(replays, func(i, j int) bool { return replays[i].Date.After(replays[j].Date) })
	return replays, errs
}
