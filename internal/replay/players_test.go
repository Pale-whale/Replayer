package replay

import (
	"slices"
	"testing"
)

func TestApplyAliases(t *testing.T) {
	r := &Replay{
		Recorder: "Alt",
		Players:  []Player{{Name: "Alt", Team: 1}, {Name: "Mate", Team: 1}, {Name: "Opp", Team: 0}},
		Goals:    []Goal{{Player: "Alt", Team: 1}},
	}
	r.ApplyAliases(map[string]string{"Alt": "Main"})
	if r.Recorder != "Main" || r.Players[0].Name != "Main" || r.Goals[0].Player != "Main" {
		t.Errorf("aliases not applied: %+v", r)
	}
	if got := r.Coached([]string{"Main", "Mate"}); !slices.Equal(got, []string{"Main", "Mate"}) || r.OurTeam([]string{"Main"}) != 1 {
		t.Errorf("coached %v", got)
	}
}
