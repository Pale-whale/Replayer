// Package trends aggregates the analysis of several replays to show how the
// coached players evolve.
package trends

import (
	"fmt"
	"math"
	"slices"
	"sort"

	"replayer/internal/analysis"
	"replayer/internal/replay"
)

// Low boost threshold for conceded goals (boost 0-100).
const lowBoost = 20

// PlayerMatch is a player's metrics in one match, plus the goals his team
// conceded and how he was placed 2 s before them.
type PlayerMatch struct {
	analysis.PlayerMetrics
	Conceded          int `json:"conceded"`
	ConcededNotBehind int `json:"conceded_not_behind_ball"`
	ConcededLowBoost  int `json:"conceded_low_boost"`
}

// ForPlayer extracts a player's metrics from a match report.
func ForPlayer(rep *analysis.Report, name string) (PlayerMatch, bool) {
	i := slices.IndexFunc(rep.Players, func(p analysis.PlayerMetrics) bool { return p.Name == name })
	if i < 0 {
		return PlayerMatch{}, false
	}
	pm := PlayerMatch{PlayerMetrics: rep.Players[i]}
	for _, g := range rep.Goals {
		if g.ScoringTeam == pm.Team {
			continue
		}
		pm.Conceded++
		for _, s := range g.Positioning {
			if s.Player != name {
				continue
			}
			if !s.BehindBall {
				pm.ConcededNotBehind++
			}
			if s.Boost < lowBoost {
				pm.ConcededLowBoost++
			}
		}
	}
	return pm, true
}

// Metric is a column shown in the TUI and averaged in the trends.
type Metric struct {
	Key   string
	Label string
	Help  string
	Get   func(PlayerMatch) float64
}

var Metrics = []Metric{
	{"avg_boost", "Boost", "boost moyen", func(p PlayerMatch) float64 { return p.AvgBoost }},
	{"zero_boost_pct", "0bst%", "temps à 0 boost", func(p PlayerMatch) float64 { return p.ZeroBoostPct }},
	{"behind_ball_pct", "Derr%", "derrière la balle", func(p PlayerMatch) float64 { return p.BehindBallPct }},
	{"defensive_third_pct", "Déf%", "tiers défensif", func(p PlayerMatch) float64 { return p.DefThirdPct }},
	{"offensive_third_pct", "Off%", "tiers offensif", func(p PlayerMatch) float64 { return p.OffThirdPct }},
	{"first_man_pct", "1er%", "1er homme", func(p PlayerMatch) float64 { return p.FirstManPct }},
	{"supersonic_pct", "SS%", "supersonique", func(p PlayerMatch) float64 { return p.SupersonicPct }},
	{"touches", "Touch", "touches", func(p PlayerMatch) float64 { return float64(p.Touches) }},
	{"kickoffs_went", "KO", "kickoffs allés", func(p PlayerMatch) float64 { return float64(p.KickoffsWent) }},
	{"kickoffs_went_won", "KO+", "kickoffs gagnés", func(p PlayerMatch) float64 { return float64(p.KickoffsWentWon) }},
	{"conceded_not_behind_ball", "BE!", "buts encaissés en étant devant la balle", func(p PlayerMatch) float64 { return float64(p.ConcededNotBehind) }},
}

type Match struct {
	ID              string                 `json:"id"`
	Date            string                 `json:"date"`
	Map             string                 `json:"map"`
	Mode            string                 `json:"mode"`
	Won             bool                   `json:"won"`
	ScoreUs         int                    `json:"score_us"`
	ScoreThem       int                    `json:"score_them"`
	KickoffsWon     int                    `json:"team_kickoffs_won"`
	KickoffsLost    int                    `json:"team_kickoffs_lost"`
	KickoffsNeutral int                    `json:"team_kickoffs_neutral"`
	Players         map[string]PlayerMatch `json:"players"`
}

type PlayerTrend struct {
	Name    string `json:"name"`
	Matches int    `json:"matches"`
	Wins    int    `json:"wins"`
	// Per Metric.Key: average over the matches, and average of the most
	// recent half minus average of the oldest half.
	Mean      map[string]float64 `json:"mean"`
	Evolution map[string]float64 `json:"evolution_recent_minus_old"`
}

type Trends struct {
	Notes   []string      `json:"notes"`
	Players []string      `json:"players"`
	Matches []Match       `json:"matches"` // oldest first
	Trends  []PlayerTrend `json:"trends"`
}

// Build aggregates the replays that have a report. players is the config
// list of coached players.
func Build(replays []*replay.Replay, reports map[string]*analysis.Report, players []string) *Trends {
	rs := slices.Clone(replays)
	sort.Slice(rs, func(i, j int) bool { return rs[i].Date.Before(rs[j].Date) })

	t := &Trends{Notes: []string{
		"Matchs du plus ancien au plus récent. Le détail de chaque match est dans matches/<date>_<id>.json (en-tête + analyse complète).",
		"mean = moyenne par match. evolution_recent_minus_old = moyenne de la moitié la plus récente des matchs moins celle de la plus ancienne (positif = en hausse).",
		"conceded = buts encaissés par l'équipe du joueur ; conceded_not_behind_ball / conceded_low_boost = parmi eux, ceux où le joueur était devant la balle / sous 20 de boost 2 s avant.",
		"team_kickoffs_* = kickoffs de l'équipe des joueurs coachés (voir les définitions dans les notes de chaque analyse).",
	}}
	for _, r := range rs {
		rep := reports[r.ID]
		if rep == nil {
			continue
		}
		us, them := r.Scores(players)
		ours := "bleu"
		if r.OurTeam(players) == 1 {
			ours = "orange"
		}
		m := Match{
			ID: r.ID, Date: r.Date.Format("2006-01-02 15:04"), Map: r.Map,
			Mode: fmt.Sprintf("%s %dv%d", r.MatchType, r.TeamSize, r.TeamSize),
			Won:  us > them, ScoreUs: us, ScoreThem: them, Players: map[string]PlayerMatch{},
		}
		for _, ko := range rep.Kickoffs {
			switch ko.Winner {
			case ours:
				m.KickoffsWon++
			case "neutre":
				m.KickoffsNeutral++
			default:
				m.KickoffsLost++
			}
		}
		for _, name := range r.Coached(players) {
			if pm, ok := ForPlayer(rep, name); ok {
				m.Players[name] = pm
				if !slices.Contains(t.Players, name) {
					t.Players = append(t.Players, name)
				}
			}
		}
		t.Matches = append(t.Matches, m)
	}

	for _, name := range t.Players {
		var ms []PlayerMatch
		pt := PlayerTrend{Name: name, Mean: map[string]float64{}, Evolution: map[string]float64{}}
		for _, m := range t.Matches {
			if pm, ok := m.Players[name]; ok {
				ms = append(ms, pm)
				if m.Won {
					pt.Wins++
				}
			}
		}
		pt.Matches = len(ms)
		half := len(ms) / 2
		for _, met := range Metrics {
			pt.Mean[met.Key] = mean(ms, met)
			if half > 0 {
				pt.Evolution[met.Key] = round(mean(ms[len(ms)-half:], met) - mean(ms[:half], met))
			}
		}
		t.Trends = append(t.Trends, pt)
	}
	return t
}

func mean(ms []PlayerMatch, met Metric) float64 {
	if len(ms) == 0 {
		return 0
	}
	sum := 0.0
	for _, m := range ms {
		sum += met.Get(m)
	}
	return round(sum / float64(len(ms)))
}

func round(x float64) float64 { return math.Round(x*10) / 10 }
