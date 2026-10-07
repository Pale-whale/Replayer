package trends

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"replayer/internal/analysis"
	"replayer/internal/replay"
)

// Runs on real replays: RL_DEMOS=<Demos dir> go test ./internal/trends -v
// Builds the trends of the last 10 2v2 played by PaleWhale with jamb0n70.
func TestBuildLocal(t *testing.T) {
	dir := os.Getenv("RL_DEMOS")
	if dir == "" {
		t.Skip("RL_DEMOS not set")
	}
	players := []string{"PaleWhale", "jamb0n70"}
	all, _ := replay.Scan(dir)
	var rs []*replay.Replay
	reports := map[string]*analysis.Report{}
	for _, r := range all {
		if r.TeamSize != 2 || !slices.Equal(r.Coached(players), players) {
			continue
		}
		rep, err := analysis.File(r, nil)
		if err != nil {
			t.Fatal(err)
		}
		rs, reports[r.ID] = append(rs, r), rep
		if len(rs) == 10 {
			break
		}
	}
	tr := Build(rs, reports, players)
	if len(tr.Matches) != 10 || !slices.Equal(tr.Players, players) {
		t.Fatalf("%d matches, players %v", len(tr.Matches), tr.Players)
	}
	for i := 1; i < len(tr.Matches); i++ {
		if tr.Matches[i].Date < tr.Matches[i-1].Date {
			t.Errorf("matches not in chronological order")
		}
	}
	for _, pt := range tr.Trends {
		if pt.Matches != 10 {
			t.Errorf("%s: %d matches", pt.Name, pt.Matches)
		}
		for _, met := range Metrics {
			if v := pt.Mean[met.Key]; v < 0 || (met.Key[len(met.Key)-3:] == "pct" && v > 100) {
				t.Errorf("%s %s = %v", pt.Name, met.Key, v)
			}
		}
	}
	b, _ := json.MarshalIndent(tr.Trends, "", "  ")
	t.Logf("%s", b)
}
