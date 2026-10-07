package analysis

import (
	"sort"

	"replayer/internal/replay"
)

const (
	fiftyWindow      = 0.3 // max seconds between the touches of a 50/50
	kickoffExclusion = 0.5 // touches this soon after the kickoff contact belong to the kickoff
	// An opposing car this close to the ball at a touch makes it a 50/50
	// (tuned against the exact BallTouches counter).
	fiftyContactDist = 200.0
)

// cluster is a run of touches less than fiftyWindow apart. It is a 50/50
// when both teams touched the ball, or when an opponent was in contact.
type cluster struct {
	touches []detectedTouch
	period  int
	fifty   bool
}

type possessionStats struct {
	inPlay, self, mate, opp, fifty, none int
	duels, duelsWon, duelsLost           int
}

// periods maps a frame to its play period: a new one starts at each kickoff.
type periods []int

func newPeriods(snaps []snapshot, kickoffStarts []int) periods {
	p := make(periods, len(kickoffStarts))
	for i, si := range kickoffStarts {
		p[i] = snaps[si].Frame
	}
	return p
}

func (p periods) of(frame int) int { return sort.SearchInts(p, frame+1) }

// inPlayTouches drops the kickoff contacts, already covered by the kickoff stats.
func inPlayTouches(touches []detectedTouch, kos []Kickoff) []detectedTouch {
	var out []detectedTouch
	for _, t := range touches {
		kickoff := false
		for _, ko := range kos {
			if t.Time >= ko.touchTime-0.1 && t.Time <= ko.touchTime+kickoffExclusion {
				kickoff = true
				break
			}
		}
		if !kickoff {
			out = append(out, t)
		}
	}
	return out
}

func clusters(ts []detectedTouch, p periods) []cluster {
	var out []cluster
	for i := 0; i < len(ts); {
		period := p.of(ts[i].Frame)
		j := i + 1
		for j < len(ts) && p.of(ts[j].Frame) == period && ts[j].Time-ts[j-1].Time <= fiftyWindow {
			j++
		}
		c := cluster{touches: ts[i:j], period: period}
		for _, t := range c.touches {
			c.fifty = c.fifty || t.Contested || t.Team != c.touches[0].Team
		}
		out = append(out, c)
		i = j
	}
	return out
}

// possession classifies each in-play touch by what follows it: a touch of
// the same player, of a teammate, of an opponent, a 50/50, or nothing (goal
// or end of the match). A 50/50 is won by the team that touches the ball
// next (or scores).
func possession(touches []detectedTouch, kos []Kickoff, p periods, goals []replay.Goal) map[string]*possessionStats {
	goalTeam := map[int]int{}
	for _, g := range goals {
		goalTeam[p.of(g.Frame)] = g.Team
	}
	stats := map[string]*possessionStats{}
	get := func(name string) *possessionStats {
		if stats[name] == nil {
			stats[name] = &possessionStats{}
		}
		return stats[name]
	}

	cs := clusters(inPlayTouches(touches, kos), p)
	for ci, c := range cs {
		var next *detectedTouch
		if ci+1 < len(cs) && cs[ci+1].period == c.period {
			next = &cs[ci+1].touches[0]
		}
		if c.fifty {
			winner, ok := goalTeam[c.period]
			if next != nil {
				winner, ok = next.Team, true
			}
			seen := map[string]bool{}
			for _, t := range c.touches {
				s := get(t.Player)
				s.inPlay++
				s.fifty++
				if seen[t.Player] {
					continue
				}
				seen[t.Player] = true
				s.duels++
				switch {
				case !ok:
				case winner == t.Team:
					s.duelsWon++
				default:
					s.duelsLost++
				}
			}
			continue
		}
		for k, t := range c.touches {
			nx := next
			if k+1 < len(c.touches) {
				nx = &c.touches[k+1]
			}
			s := get(t.Player)
			s.inPlay++
			switch {
			case nx == nil:
				s.none++
			case nx.Player == t.Player:
				s.self++
			case nx.Team == t.Team:
				s.mate++
			default:
				s.opp++
			}
		}
	}
	return stats
}
