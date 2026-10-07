package replay

import (
	"os"
	"testing"
)

func TestScanLocal(t *testing.T) {
	dir := os.Getenv("RL_DEMOS")
	if dir == "" {
		t.Skip("RL_DEMOS not set")
	}
	rs, errs := Scan(dir)
	for _, e := range errs {
		t.Error(e)
	}
	t.Logf("%d parsed", len(rs))
	for _, r := range rs[:3] {
		t.Logf("%s %s %s %d-%d %+v goals=%d", r.Date.Format("2006-01-02 15:04"), r.Map, r.MatchType, r.Team0Score, r.Team1Score, r.Players, len(r.Goals))
	}
}
