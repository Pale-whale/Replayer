package analysis

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"replayer/internal/replay"
)

// Runs on real replays: RL_DEMOS=<Demos dir> go test ./internal/analysis -v
func demos(t *testing.T) []*replay.Replay {
	dir := os.Getenv("RL_DEMOS")
	if dir == "" {
		t.Skip("RL_DEMOS not set")
	}
	rs, _ := replay.Scan(dir)
	return rs
}

// Compares the touch heuristic with the exact BallTouches counter that
// recent replays carry.
func TestTouchHeuristic(t *testing.T) {
	var exact, found, matched, good, n int
	for _, r := range demos(t) {
		nr, err := decode(r.Path)
		if err != nil {
			t.Fatal(err)
		}
		tl := buildTimeline(nr)
		if len(tl.Touches) == 0 {
			continue
		}
		det := detectTouches(tl.Snaps)
		n++
		exact += len(tl.Touches)
		found += len(det)
		near := func(d detectedTouch, e touch) bool { return d.Player == e.Player && math.Abs(d.Time-e.Time) < 0.2 }
		for _, e := range tl.Touches {
			if slices.ContainsFunc(det, func(d detectedTouch) bool { return near(d, e) }) {
				matched++
			}
		}
		for _, d := range det {
			if slices.ContainsFunc(tl.Touches, func(e touch) bool { return near(d, e) }) {
				good++
			}
		}
	}
	t.Logf("%d replays: exact=%d detected=%d (recall %.0f%%, precision %.0f%%)",
		n, exact, found, 100*float64(matched)/float64(exact), 100*float64(good)/float64(found))
}

func TestAnalyzeAll(t *testing.T) {
	for i, r := range demos(t) {
		nr, err := decode(r.Path)
		if err != nil {
			t.Fatalf("%s: %v", filepath.Base(r.Path), err)
		}
		rep := analyze(r, nr)
		if len(rep.Players) == 0 || rep.InPlaySeconds < 20 {
			t.Errorf("%s (%s): %d players, %.0fs in play", filepath.Base(r.Path), r.Date, len(rep.Players), rep.InPlaySeconds)
		}
		if i == 0 && testing.Verbose() {
			b, _ := json.MarshalIndent(rep, "", "  ")
			t.Logf("%s", b)
		}
	}
}

// Checks the kickoff detection: one kickoff per goal + the opening one,
// realistic time to ball and someone going for each team.
func TestKickoffs(t *testing.T) {
	var total, fast, bothWent, badCount int
	winners := map[string]int{}
	for i, r := range demos(t) {
		nr, err := decode(r.Path)
		if err != nil {
			t.Fatal(err)
		}
		rep := analyze(r, nr)
		if d := len(rep.Kickoffs) - (len(r.Goals) + 1); d < -1 || d > 0 {
			badCount++
			t.Logf("%s: %d kickoffs for %d goals", filepath.Base(r.Path), len(rep.Kickoffs), len(r.Goals))
		}
		for _, ko := range rep.Kickoffs {
			total++
			winners[ko.Winner]++
			if ko.TimeToBall >= 1.8 && ko.TimeToBall <= 4 {
				fast++
			}
			var went [2]bool
			for _, p := range ko.Players {
				went[p.Team] = went[p.Team] || p.Went
			}
			if went[0] && went[1] {
				bothWent++
			}
		}
		if i == 0 && testing.Verbose() {
			b, _ := json.MarshalIndent(rep.Kickoffs, "", "  ")
			t.Logf("%s", b)
		}
	}
	t.Logf("%d kickoffs: %.0f%% time_to_ball in 1.8-4 s, %.0f%% both teams went, winners %v, %d replays with odd count",
		total, 100*float64(fast)/float64(total), 100*float64(bothWent)/float64(total), winners, badCount)
	if badCount > 5 {
		t.Errorf("%d replays with unexpected kickoff count", badCount)
	}
}

// Every in-play touch falls in exactly one possession category.
func TestPossessionSums(t *testing.T) {
	for _, r := range demos(t) {
		nr, err := decode(r.Path)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range analyze(r, nr).Players {
			if p.TouchesInPlay == 0 {
				continue
			}
			sum := p.FollowedSelfPct + p.FollowedMatePct + p.GivenPct + p.FiftyPct + p.UnfollowedPct
			if math.Abs(sum-100) > 0.6 || math.Abs(p.KeptPct-p.FollowedSelfPct-p.FollowedMatePct) > 0.2 {
				t.Errorf("%s %s: categories sum to %.1f", filepath.Base(r.Path), p.Name, sum)
			}
			if p.FiftyWon+p.FiftyLost > p.FiftyFifties {
				t.Errorf("%s %s: 50/50 outcomes", filepath.Base(r.Path), p.Name)
			}
		}
	}
}

// Compares the detected 50/50s with the ones of the exact BallTouches
// counter: touches of both teams less than 0.3 s apart.
func TestFiftyFifty(t *testing.T) {
	var exact, found, matched, good int
	for _, r := range demos(t) {
		nr, err := decode(r.Path)
		if err != nil {
			t.Fatal(err)
		}
		tl := buildTimeline(nr)
		if len(tl.Touches) == 0 {
			continue
		}
		rep := analyze(r, nr)
		team := map[string]int{}
		for _, p := range rep.Players {
			team[p.Name] = p.Team
		}
		var exactTouches []detectedTouch
		for _, e := range tl.Touches {
			exactTouches = append(exactTouches, detectedTouch{touch: e, Team: team[e.Player]})
		}
		p := newPeriods(tl.Snaps, tl.KickoffStarts)
		duelTimes := func(ts []detectedTouch, useContact bool) []float64 {
			var out []float64
			for _, c := range clusters(inPlayTouches(ts, rep.Kickoffs), p) {
				both := false
				for _, x := range c.touches {
					both = both || x.Team != c.touches[0].Team || (useContact && x.Contested)
				}
				if both {
					out = append(out, c.touches[0].Time)
				}
			}
			return out
		}
		ex, det := duelTimes(exactTouches, false), duelTimes(detectTouches(tl.Snaps), true)
		near := func(a float64, bs []float64) bool {
			return slices.ContainsFunc(bs, func(b float64) bool { return math.Abs(a-b) < 0.4 })
		}
		exact, found = exact+len(ex), found+len(det)
		for _, e := range ex {
			if near(e, det) {
				matched++
			}
		}
		for _, d := range det {
			if near(d, ex) {
				good++
			}
		}
	}
	t.Logf("50/50: exact=%d detected=%d (recall %.0f%%, precision %.0f%%)",
		exact, found, 100*float64(matched)/float64(exact), 100*float64(good)/float64(found))
}
