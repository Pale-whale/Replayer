package analysis

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	"replayer/internal/rank"
	"replayer/internal/replay"
)

// Field geometry, in Unreal units (1 uu = 1 cm). Blue (team 0) defends y < 0.
const (
	thirdY        = 5120.0 / 3
	supersonic    = 2200.0
	groundZ       = 40.0  // car resting height is ~17
	highAirZ      = 300.0 // above this the car is high in the air or up a wall
	touchMinDV    = 200.0 // ball velocity change (uu/s) between two frames
	touchMaxDist  = 260.0 // car center to ball center
	goalLookback  = 2.0   // seconds before a goal for the positioning snapshot
	touchLookback = 10.0  // seconds before a goal for the touch sequence
)

// Version of the report format: cached reports of another version are
// recomputed.
const Version = 2

type Report struct {
	Version       int             `json:"version"`
	AliasesKey    string          `json:"aliases_key,omitempty"`
	Playlist      string          `json:"playlist"`
	Ranked        bool            `json:"ranked"`
	Notes         []string        `json:"notes"`
	InPlaySeconds float64         `json:"in_play_seconds"`
	Players       []PlayerMetrics `json:"players"`
	Goals         []GoalContext   `json:"goals"`
	Kickoffs      []Kickoff       `json:"kickoffs"`
	Demos         []Demo          `json:"demos"`
}

type PlayerMetrics struct {
	Name          string  `json:"name"`
	Team          int     `json:"team"`
	InPlaySeconds float64 `json:"in_play_seconds"`

	DefThirdPct   float64 `json:"defensive_third_pct"`
	MidThirdPct   float64 `json:"middle_third_pct"`
	OffThirdPct   float64 `json:"offensive_third_pct"`
	BehindBallPct float64 `json:"behind_ball_pct"`
	AvgBallDist   float64 `json:"avg_distance_to_ball"`
	// Only meaningful with teammates: closest teammate to the ball, and the
	// one closest to his own goal.
	FirstManPct float64 `json:"first_man_pct,omitempty"`
	LastManPct  float64 `json:"last_man_pct,omitempty"`

	AvgSpeed      float64 `json:"avg_speed"`
	SupersonicPct float64 `json:"supersonic_pct"`
	GroundPct     float64 `json:"ground_pct"`
	LowAirPct     float64 `json:"low_air_or_wall_pct"`
	HighAirPct    float64 `json:"high_air_or_wall_pct"`

	AvgBoost      float64 `json:"avg_boost"`
	ZeroBoostPct  float64 `json:"zero_boost_pct"`
	LowBoostPct   float64 `json:"under_25_boost_pct"`
	FullBoostPct  float64 `json:"over_80_boost_pct"`
	BigPads       int     `json:"big_pads"`
	SmallPads     int     `json:"small_pads"`
	AvgBoostAtBig float64 `json:"avg_boost_before_big_pad"`

	Touches       int `json:"touches"`
	TouchesDef    int `json:"touches_defensive_third"`
	TouchesMid    int `json:"touches_middle_third"`
	TouchesOff    int `json:"touches_offensive_third"`
	DemosDone     int `json:"demos_inflicted"`
	DemosReceived int `json:"demos_received"`

	Kickoffs         int `json:"kickoffs"`
	KickoffsWent     int `json:"kickoffs_went"`
	KickoffsWentWon  int `json:"kickoffs_went_won"`
	KickoffsWentLost int `json:"kickoffs_went_lost"`

	// Possession: what follows each in-play touch (kickoffs excluded).
	TouchesInPlay   int     `json:"touches_in_play"`
	FollowedSelfPct float64 `json:"touches_followed_by_self_pct"`
	FollowedMatePct float64 `json:"touches_followed_by_teammate_pct"`
	KeptPct         float64 `json:"touches_kept_by_team_pct"`
	GivenPct        float64 `json:"touches_given_to_opponent_pct"`
	FiftyPct        float64 `json:"fifty_fifty_pct"`
	UnfollowedPct   float64 `json:"touches_unfollowed_pct"`
	FiftyFifties    int     `json:"fifty_fifties"`
	FiftyWon        int     `json:"fifty_fifty_won"`
	FiftyLost       int     `json:"fifty_fifty_lost"`
}

type GoalContext struct {
	Clock       string        `json:"clock"`
	Scorer      string        `json:"scorer"`
	ScoringTeam int           `json:"scoring_team"`
	LastTouches []TouchEvent  `json:"last_touches"`
	Positioning []PlayerState `json:"positioning_2s_before"`
	MissingCars []string      `json:"demolished_or_respawning_2s_before,omitempty"`
	BallHeight  int           `json:"ball_height_pct_2s_before"`
	// Field seen from above 2 s before the goal, then its legend.
	FieldMap []string `json:"field_map_2s_before,omitempty"`
}

type TouchEvent struct {
	Player        string  `json:"player"`
	Team          int     `json:"team"`
	SecondsBefore float64 `json:"seconds_before_goal"`
	// Zone of the ball, from the toucher's point of view.
	Zone string `json:"zone"`
}

type PlayerState struct {
	Player     string `json:"player"`
	Label      string `json:"map_label"`
	Team       int    `json:"team"`
	Zone       string `json:"zone"`
	BallDist   int    `json:"distance_to_ball"`
	Boost      int    `json:"boost"`
	BehindBall bool   `json:"behind_ball"`
	Speed      int    `json:"speed"`
	Height     int    `json:"height_pct"`
}

type Demo struct {
	Clock    string `json:"clock"`
	Attacker string `json:"attacker"`
	Victim   string `json:"victim"`
}

// File decodes the replay with rrrocket and builds its coaching report.
// aliases renames players (alias -> main name); r must already have them
// applied (replay.ApplyAliases).
func File(r *replay.Replay, aliases map[string]string) (*Report, error) {
	nr, err := decode(r.Path)
	if err != nil {
		return nil, err
	}
	nr.aliases = aliases
	rep := analyze(r, nr)
	rep.AliasesKey = AliasesKey(aliases)
	return rep, nil
}

// AliasesKey identifies a set of aliases: a cached report made with other
// aliases has other player names and must be recomputed.
func AliasesKey(aliases map[string]string) string {
	var pairs []string
	for a, n := range aliases {
		pairs = append(pairs, a+"="+n)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ";")
}

func alias(aliases map[string]string, name string) string {
	if n, ok := aliases[name]; ok {
		return n
	}
	return name
}

func analyze(r *replay.Replay, nr *netReplay) *Report {
	tl := buildTimeline(nr)
	touches := detectTouches(tl.Snaps)

	rep := &Report{Version: Version, Playlist: rank.PlaylistName(tl.Playlist), Ranked: rank.Ranked(tl.Playlist), Notes: []string{
		"Distances en uu (1 uu = 1 cm), vitesses en uu/s (max 2300, supersonique >= 2200). Boost en % (0-100).",
		"Les pourcentages sont calculés sur le temps de jeu actif (hors compte à rebours et célébrations de but).",
		"Zones (tiers défensif/milieu/offensif) toujours du point de vue du joueur concerné. team 0 = bleu, team 1 = orange.",
		"Le boost entre deux mises à jour réseau est simulé (conso 33 %/s) : valeurs approximatives à quelques % près.",
		"Les touches de balle sont détectées par heuristique (changement de vitesse de la balle + voiture à proximité) : quelques erreurs possibles.",
		"low_air_or_wall / high_air_or_wall : la hauteur ne distingue pas le vol du roulage au mur.",
		"Kickoffs : went = le joueur de l'équipe le plus proche de la balle à la 1re touche (personne si > 800 uu : fake). winner = équipe qui marque dans les 3 s après la 1re touche, sinon moitié de terrain où est la balle 3 s après (neutre à moins de 1000 uu du milieu). time_to_ball = secondes entre le top départ et la 1re touche (la balle quitte le centre). En prolongation, clock = temps écoulé depuis le début de la prolongation.",
		"Possession (touches en jeu = hors contact de kickoff) : chaque touche est classée selon la touche suivante : même joueur (followed_by_self), coéquipier (followed_by_teammate ; kept_by_team = les deux), adversaire (given_to_opponent : balle rendue), ou aucune (unfollowed : but ou fin). Une touche est un 50/50 si les deux équipes touchent la balle à moins de 0,3 s d'intervalle ou si un adversaire est au contact ; le 50/50 est gagné par l'équipe qui touche la balle ensuite (ou qui marque).",
		"Hauteurs en % du plafond (2044 uu ; approximatif en hoops). field_map : terrain vu de dessus, but bleu à gauche ; 1 2 3 = bleus, A B C = orange, o = balle ; : = limites des tiers, | = milieu.",
	}}

	stats := map[string]*PlayerMetrics{}
	acc := map[string]*accum{}
	get := func(c carSnap) (*PlayerMetrics, *accum) {
		if stats[c.Player] == nil {
			stats[c.Player] = &PlayerMetrics{Name: c.Player, Team: c.Team}
			acc[c.Player] = &accum{}
		}
		return stats[c.Player], acc[c.Player]
	}

	for _, s := range tl.Snaps {
		dt := min(s.Delta, 0.2) // guard against time jumps
		rep.InPlaySeconds += dt
		first, last := roles(s)
		for _, c := range s.Cars {
			m, a := get(c)
			m.InPlaySeconds += dt
			ny, by := norm(c.Team, c.Pos.Y), norm(c.Team, s.Ball.Y)
			switch zone(ny) {
			case "def":
				a.def += dt
			case "mid":
				a.mid += dt
			default:
				a.off += dt
			}
			if ny < by {
				a.behind += dt
			}
			a.dist += dist(c.Pos, s.Ball) * dt
			if first[c.Team] == c.Player {
				a.first += dt
			}
			if last[c.Team] == c.Player {
				a.last += dt
			}
			if teamSize(s, c.Team) > 1 {
				a.withMates += dt
			}
			speed := norm3(c.Vel)
			a.speed += speed * dt
			if speed >= supersonic {
				a.supersonic += dt
			}
			switch {
			case c.Pos.Z < groundZ:
				a.ground += dt
			case c.Pos.Z < highAirZ:
				a.lowAir += dt
			default:
				a.highAir += dt
			}
			a.boost += c.Boost * dt
			if c.Boost < 1 {
				a.zero += dt
			}
			if c.Boost < 25 {
				a.low += dt
			}
			if c.Boost > 80 {
				a.full += dt
			}
		}
	}

	for _, p := range tl.Pickups {
		m := stats[p.Player]
		if m == nil {
			continue
		}
		if p.Big {
			m.BigPads++
			acc[p.Player].boostAtBig += p.Before
		} else {
			m.SmallPads++
		}
	}
	for _, t := range touches {
		m := stats[t.Player]
		if m == nil {
			continue
		}
		m.Touches++
		switch zone(norm(m.Team, t.Ball.Y)) {
		case "def":
			m.TouchesDef++
		case "mid":
			m.TouchesMid++
		default:
			m.TouchesOff++
		}
	}
	for _, d := range tl.Demos {
		if m := stats[d.Attacker]; m != nil {
			m.DemosDone++
		}
		if m := stats[d.Victim]; m != nil {
			m.DemosReceived++
		}
		rep.Demos = append(rep.Demos, Demo{Clock: fmtClock(d.Clock, d.Overtime), Attacker: d.Attacker, Victim: d.Victim})
	}

	rep.Kickoffs = kickoffs(tl.Snaps, tl.KickoffStarts, r.Goals)
	for _, ko := range rep.Kickoffs {
		for _, p := range ko.Players {
			m := stats[p.Player]
			if m == nil {
				continue
			}
			m.Kickoffs++
			if !p.Went {
				continue
			}
			m.KickoffsWent++
			switch ko.Winner {
			case teamName(p.Team):
				m.KickoffsWentWon++
			case teamName(1 - p.Team):
				m.KickoffsWentLost++
			}
		}
	}

	poss := possession(touches, rep.Kickoffs, newPeriods(tl.Snaps, tl.KickoffStarts), r.Goals)
	for name, ps := range poss {
		m := stats[name]
		if m == nil || ps.inPlay == 0 {
			continue
		}
		n := float64(ps.inPlay)
		m.TouchesInPlay = ps.inPlay
		m.FollowedSelfPct, m.FollowedMatePct = pct(float64(ps.self), n), pct(float64(ps.mate), n)
		m.KeptPct = pct(float64(ps.self+ps.mate), n)
		m.GivenPct, m.FiftyPct, m.UnfollowedPct = pct(float64(ps.opp), n), pct(float64(ps.fifty), n), pct(float64(ps.none), n)
		m.FiftyFifties, m.FiftyWon, m.FiftyLost = ps.duels, ps.duelsWon, ps.duelsLost
	}

	for name, m := range stats {
		a := acc[name]
		t := m.InPlaySeconds
		if t == 0 {
			continue
		}
		m.DefThirdPct, m.MidThirdPct, m.OffThirdPct = pct(a.def, t), pct(a.mid, t), pct(a.off, t)
		m.BehindBallPct = pct(a.behind, t)
		m.AvgBallDist = round(a.dist / t)
		if a.withMates > 0 {
			m.FirstManPct, m.LastManPct = pct(a.first, a.withMates), pct(a.last, a.withMates)
		}
		m.AvgSpeed = round(a.speed / t)
		m.SupersonicPct = pct(a.supersonic, t)
		m.GroundPct, m.LowAirPct, m.HighAirPct = pct(a.ground, t), pct(a.lowAir, t), pct(a.highAir, t)
		m.AvgBoost = round(a.boost / t)
		m.ZeroBoostPct, m.LowBoostPct, m.FullBoostPct = pct(a.zero, t), pct(a.low, t), pct(a.full, t)
		if m.BigPads > 0 {
			m.AvgBoostAtBig = round(a.boostAtBig / float64(m.BigPads))
		}
		m.InPlaySeconds = round(t)
		rep.Players = append(rep.Players, *m)
	}
	sort.Slice(rep.Players, func(i, j int) bool {
		if rep.Players[i].Team != rep.Players[j].Team {
			return rep.Players[i].Team < rep.Players[j].Team
		}
		return rep.Players[i].Name < rep.Players[j].Name
	})
	rep.InPlaySeconds = round(rep.InPlaySeconds)

	for _, g := range r.Goals {
		rep.Goals = append(rep.Goals, goalContext(g, tl.Snaps, touches))
	}
	return rep
}

type accum struct {
	def, mid, off, behind, dist, first, last, withMates float64
	speed, supersonic, ground, lowAir, highAir          float64
	boost, zero, low, full, boostAtBig                  float64
}

type detectedTouch struct {
	touch
	Team int
	Ball vec
	// An opposing car was also within touch distance of the ball.
	Contested bool
}

// detectTouches finds ball touches: a sudden change of the ball velocity
// between two consecutive frames with a car close to the ball.
func detectTouches(snaps []snapshot) []detectedTouch {
	var out []detectedTouch
	for i := 1; i < len(snaps); i++ {
		a, b := snaps[i-1], snaps[i]
		if b.Frame != a.Frame+1 || dist(a.BallVel, b.BallVel) < touchMinDV {
			continue
		}
		best, bestDist := -1, math.Inf(1)
		for ci, c := range b.Cars {
			if d := dist(c.Pos, b.Ball); d < bestDist {
				best, bestDist = ci, d
			}
		}
		if best < 0 || bestDist > touchMaxDist {
			continue
		}
		c := b.Cars[best]
		if n := len(out); n > 0 && out[n-1].Player == c.Player && b.Time-out[n-1].Time < 0.25 {
			continue // same contact spread over several frames
		}
		contested := slices.ContainsFunc(b.Cars, func(o carSnap) bool { return o.Team != c.Team && dist(o.Pos, b.Ball) <= fiftyContactDist })
		out = append(out, detectedTouch{touch: touch{Frame: b.Frame, Time: b.Time, Player: c.Player}, Team: c.Team, Ball: b.Ball, Contested: contested})
	}
	return out
}

func goalContext(g replay.Goal, snaps []snapshot, touches []detectedTouch) GoalContext {
	gc := GoalContext{Scorer: g.Player, ScoringTeam: g.Team}
	// last snapshot at or before the goal frame
	i := sort.Search(len(snaps), func(i int) bool { return snaps[i].Frame > g.Frame }) - 1
	if i < 0 {
		return gc
	}
	goal := snaps[i]
	gc.Clock = fmtClock(goal.Clock, goal.Overtime)

	for j := len(touches) - 1; j >= 0 && len(gc.LastTouches) < 4; j-- {
		t := touches[j]
		if t.Frame > goal.Frame {
			continue
		}
		if goal.Time-t.Time > touchLookback {
			break
		}
		gc.LastTouches = append(gc.LastTouches, TouchEvent{
			Player: t.Player, Team: t.Team, SecondsBefore: round(goal.Time - t.Time), Zone: zoneName(zone(norm(t.Team, t.Ball.Y))),
		})
	}

	k := i
	for k > 0 && goal.Time-snaps[k].Time < goalLookback {
		k--
	}
	s := snaps[k]
	present := map[string]bool{}
	labels := mapLabels(s.Cars)
	for _, c := range s.Cars {
		present[c.Player] = true
		gc.Positioning = append(gc.Positioning, PlayerState{
			Player: c.Player, Label: labels[c.Player], Team: c.Team, Zone: zoneName(zone(norm(c.Team, c.Pos.Y))),
			BallDist: int(dist(c.Pos, s.Ball)), Boost: int(math.Round(c.Boost)),
			BehindBall: norm(c.Team, c.Pos.Y) < norm(c.Team, s.Ball.Y), Speed: int(norm3(c.Vel)),
			Height: heightPct(c.Pos.Z),
		})
	}
	gc.BallHeight = heightPct(s.Ball.Z)
	gc.FieldMap = fieldMap(s, labels)
	sort.Slice(gc.Positioning, func(a, b int) bool {
		pa, pb := gc.Positioning[a], gc.Positioning[b]
		return pa.Team < pb.Team || pa.Team == pb.Team && pa.Player < pb.Player
	})
	for _, c := range goal.Cars {
		if !present[c.Player] {
			gc.MissingCars = append(gc.MissingCars, c.Player)
		}
	}
	for _, c := range snaps[max(0, k-30)].Cars { // players demolished around the snapshot
		if !present[c.Player] && !slices.Contains(gc.MissingCars, c.Player) {
			gc.MissingCars = append(gc.MissingCars, c.Player)
		}
	}
	return gc
}

// roles returns, per team, the player closest to the ball and the one
// closest to his own goal.
func roles(s snapshot) (first, last [2]string) {
	bestD := [2]float64{math.Inf(1), math.Inf(1)}
	bestY := [2]float64{math.Inf(1), math.Inf(1)}
	for _, c := range s.Cars {
		t := c.Team
		if d := dist(c.Pos, s.Ball); d < bestD[t] {
			bestD[t], first[t] = d, c.Player
		}
		if y := norm(t, c.Pos.Y); y < bestY[t] {
			bestY[t], last[t] = y, c.Player
		}
	}
	return first, last
}

func teamSize(s snapshot, team int) int {
	n := 0
	for _, c := range s.Cars {
		if c.Team == team {
			n++
		}
	}
	return n
}

// norm returns y so that the team's own goal is at negative y.
func norm(team int, y float64) float64 {
	if team == 1 {
		return -y
	}
	return y
}

func zone(ny float64) string {
	switch {
	case ny < -thirdY:
		return "def"
	case ny > thirdY:
		return "off"
	}
	return "mid"
}

func zoneName(z string) string {
	return map[string]string{"def": "tiers défensif", "mid": "milieu", "off": "tiers offensif"}[z]
}

func fmtClock(sec int, overtime bool) string {
	if overtime {
		return "+" + mmss(sec) + " (prolongation)"
	}
	return mmss(sec)
}

func mmss(sec int) string { return fmt.Sprintf("%d:%02d", sec/60, sec%60) }

func dist(a, b vec) float64           { return norm3(vec{a.X - b.X, a.Y - b.Y, a.Z - b.Z}) }
func norm3(v vec) float64             { return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z) }
func pct(part, total float64) float64 { return round(part / total * 100) }
func round(x float64) float64         { return math.Round(x*10) / 10 }
