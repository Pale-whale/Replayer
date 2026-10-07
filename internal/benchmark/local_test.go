package benchmark

import (
	"os"
	"testing"

	"replayer/internal/analysis"
	"replayer/internal/replay"
)

// Runs on real replays: RL_DEMOS=<Demos dir> go test ./internal/benchmark -v
func TestLocal(t *testing.T) {
	dir := os.Getenv("RL_DEMOS")
	if dir == "" {
		t.Skip("RL_DEMOS not set")
	}
	all, _ := replay.Scan(dir)
	reports := map[string]*analysis.Report{}
	for _, r := range all {
		if rep, err := analysis.File(r, nil); err == nil {
			reports[r.ID] = rep
		}
	}
	b := Local(all, reports, []string{"PaleWhale", "jamb0n70"}, "ranked-doubles", 30)
	t.Logf("%d replays, %d players", b.Replays, b.Players)
	if b.Players < 60 {
		t.Errorf("only %d players", b.Players)
	}
	for _, k := range []string{"avg_boost", "behind_ball_pct", "touches", "touches_kept_by_team_pct", "fifty_fifty_pct"} {
		st, ok := b.Stat(k)
		t.Logf("%s: %+v", k, st)
		if !ok || st.P25 > st.Median || st.Median > st.P75 {
			t.Errorf("%s: %+v", k, st)
		}
	}
	if st, _ := b.Stat("avg_boost"); st.Median < 30 || st.Median > 60 {
		t.Errorf("implausible boost median %v", st.Median)
	}
}
