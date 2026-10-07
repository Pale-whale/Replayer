package analysis

import (
	"strings"
	"testing"
)

func TestFieldMap(t *testing.T) {
	s := snapshot{
		Ball: vec{0, 0, 1022},
		Cars: []carSnap{
			{Player: "bleu", Team: 0, Pos: vec{0, -5000, 17}, Boost: 34},
			{Player: "orange", Team: 1, Pos: vec{-4000, 5000, 2044}, Boost: 100},
		},
	}
	lines := fieldMap(s, mapLabels(s.Cars))
	t.Log("\n" + strings.Join(lines, "\n"))
	if len(lines) != MapRows+3 {
		t.Fatalf("%d lines", len(lines))
	}
	mid := lines[2+mapRows/2]
	if !strings.HasPrefix(mid, " [|1") || !strings.HasSuffix(mid, "|]") {
		t.Errorf("blue player should be at the left of the goal row: %q", mid)
	}
	if strings.IndexRune(mid, 'o') != 3+mapCols/2 {
		t.Errorf("ball should be at the center: %q", mid)
	}
	if last := lines[2+mapRows-1]; !strings.Contains(last, "A|") {
		t.Errorf("orange player should be at the bottom right: %q", last)
	}
	if !strings.Contains(lines[len(lines)-1], "hauteur  50%") || !strings.Contains(lines[len(lines)-2], "hauteur 100%") {
		t.Errorf("heights: %q", lines[len(lines)-2:])
	}
}
