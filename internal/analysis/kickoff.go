package analysis

import (
	"math"
	"sort"

	"replayer/internal/replay"
)

const (
	kickoffGoerMaxDist = 800.0  // farther than this at the first touch: the team faked
	kickoffOutcomeAt   = 3.0    // seconds after the first touch
	kickoffNeutralY    = 1000.0 // ball closer than this to midfield: neutral
	kickoffGoalWindow  = 10.0
	kickoffBallMoved   = 100.0 // ball speed (uu/s) meaning it has been touched
)

type Kickoff struct {
	Clock      string  `json:"clock"`
	Winner     string  `json:"winner"` // bleu, orange or neutre
	TimeToBall float64 `json:"time_to_ball"`
	// Team that scored within 10 s of the first touch, if any.
	GoalWithin10s string          `json:"goal_within_10s,omitempty"`
	Players       []KickoffPlayer `json:"players"`

	touchTime float64 // replay time of the first touch
}

type KickoffPlayer struct {
	Player string `json:"player"`
	Team   int    `json:"team"`
	Spawn  string `json:"spawn"` // diagonale, décalé or fond
	Went   bool   `json:"went"`
}

func teamName(team int) string {
	if team == 0 {
		return "bleu"
	}
	return "orange"
}

func kickoffs(snaps []snapshot, starts []int, goals []replay.Goal) []Kickoff {
	type goalAt struct {
		time float64
		team int
	}
	var gs []goalAt
	for _, g := range goals {
		if i := sort.Search(len(snaps), func(i int) bool { return snaps[i].Frame > g.Frame }) - 1; i >= 0 {
			gs = append(gs, goalAt{snaps[i].Time, g.Team})
		}
	}

	var out []Kickoff
	for k, si := range starts {
		end := len(snaps)
		if k+1 < len(starts) {
			end = starts[k+1]
		}
		start := snaps[si]
		// The ball waits at the center until the first touch (in Hoops it
		// is launched straight up, hence the horizontal speed only).
		ai := si
		for ai < end && math.Hypot(snaps[ai].BallVel.X, snaps[ai].BallVel.Y) < kickoffBallMoved {
			ai++
		}
		if ai == end {
			continue // nobody touched the ball (forfeit, end of match)
		}
		at := snaps[ai]
		ko := Kickoff{Clock: fmtClock(start.Clock, start.Overtime), TimeToBall: round(at.Time - start.Time), touchTime: at.Time}

		var goer [2]string
		best := [2]float64{kickoffGoerMaxDist, kickoffGoerMaxDist}
		for _, c := range at.Cars {
			if d := dist(c.Pos, at.Ball); d < best[c.Team] {
				best[c.Team], goer[c.Team] = d, c.Player
			}
		}
		for _, c := range start.Cars {
			ko.Players = append(ko.Players, KickoffPlayer{Player: c.Player, Team: c.Team, Spawn: spawn(c.Pos), Went: goer[c.Team] == c.Player})
		}
		sort.Slice(ko.Players, func(a, b int) bool {
			pa, pb := ko.Players[a], ko.Players[b]
			return pa.Team < pb.Team || pa.Team == pb.Team && pa.Player < pb.Player
		})

		for _, g := range gs {
			if dt := g.time - at.Time; dt >= 0 && dt <= kickoffGoalWindow {
				ko.GoalWithin10s = teamName(g.team)
				if dt <= kickoffOutcomeAt {
					ko.Winner = teamName(g.team)
				}
				break
			}
		}
		if ko.Winner == "" {
			j := si + sort.Search(end-si, func(i int) bool { return snaps[si+i].Time >= at.Time+kickoffOutcomeAt })
			y := snaps[min(j, end-1)].Ball.Y
			switch {
			case math.Abs(y) < kickoffNeutralY:
				ko.Winner = "neutre"
			case y > 0: // ball in the orange half
				ko.Winner = teamName(0)
			default:
				ko.Winner = teamName(1)
			}
		}
		out = append(out, ko)
	}
	return out
}

// spawn names the kickoff spawn position from the car location.
func spawn(p vec) string {
	switch x := math.Abs(p.X); {
	case math.Abs(p.Y) < 2000:
		return "?" // position not replicated yet
	case x > 1500:
		return "diagonale"
	case x > 100:
		return "décalé"
	}
	return "fond"
}
