// Package benchmark compares the coached players with players of the same
// level: either the other players of their recent ranked matches (local),
// or replays of the configured rank downloaded from ballchasing.com. Both
// go through our own analysis, so the metrics have the same definitions.
package benchmark

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	"replayer/internal/analysis"
	"replayer/internal/replay"
	"replayer/internal/trends"
)

const (
	// Matches shorter than this (forfeits) and players who played less
	// than minPlayerSeconds are left out of the samples.
	minMatchSeconds  = 120
	minPlayerSeconds = 60
	// Counters are scaled to this duration of play to compare matches of
	// different lengths.
	countBase = 300.0
)

type Benchmark struct {
	Source   string               `json:"source"` // "local" or "ballchasing"
	Playlist string               `json:"playlist"`
	Rank     string               `json:"rank,omitempty"`
	Built    string               `json:"built"`
	Replays  int                  `json:"replays"`
	Players  int                  `json:"players"`
	Samples  map[string][]float64 `json:"samples"`
}

type Stat struct {
	Mean   float64 `json:"mean"`
	P25    float64 `json:"p25"`
	Median float64 `json:"median"`
	P75    float64 `json:"p75"`
}

// Label names the benchmark in the TUI.
func (b *Benchmark) Label() string {
	if b.Source == "ballchasing" {
		return "Réf. " + b.Rank
	}
	return "Réf. locale"
}

func (b *Benchmark) Stat(key string) (Stat, bool) {
	s := slices.Clone(b.Samples[key])
	if len(s) == 0 {
		return Stat{}, false
	}
	sort.Float64s(s)
	q := func(p float64) float64 { return round(s[int(math.Round(p*float64(len(s)-1)))]) }
	sum := 0.0
	for _, v := range s {
		sum += v
	}
	return Stat{Mean: round(sum / float64(len(s))), P25: q(0.25), Median: q(0.5), P75: q(0.75)}, true
}

// Percentile is the share of the benchmark players below v (ties count half).
func (b *Benchmark) Percentile(key string, v float64) float64 {
	s := b.Samples[key]
	if len(s) == 0 {
		return 0
	}
	below := 0.0
	for _, x := range s {
		switch {
		case x < v:
			below++
		case x == v:
			below += 0.5
		}
	}
	return math.Round(below / float64(len(s)) * 100)
}

// Comparison is a player's value of a metric against the benchmark.
type Comparison struct {
	Value      float64 `json:"value"`
	Percentile float64 `json:"percentile"`
	Stat
}

func (b *Benchmark) Compare(values map[string]float64) map[string]Comparison {
	out := map[string]Comparison{}
	for k, v := range values {
		if st, ok := b.Stat(k); ok {
			out[k] = Comparison{Value: round(v), Percentile: b.Percentile(k, v), Stat: st}
		}
	}
	return out
}

// Values flattens a player's match into comparable values: every numeric
// metric, the counters scaled to 5 minutes of play.
func Values(pm trends.PlayerMatch) map[string]float64 {
	out := map[string]float64{}
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		t := v.Type()
		for i := range t.NumField() {
			f, fv := t.Field(i), v.Field(i)
			if f.Anonymous {
				walk(fv)
				continue
			}
			key := strings.Split(f.Tag.Get("json"), ",")[0]
			switch {
			case key == "team" || key == "in_play_seconds":
			case fv.Kind() == reflect.Float64:
				out[key] = fv.Float()
			case fv.Kind() == reflect.Int && pm.InPlaySeconds > 0:
				out[key] = float64(fv.Int()) * countBase / pm.InPlaySeconds
			}
		}
	}
	walk(reflect.ValueOf(pm))
	return out
}

// MeanValues averages Values over several matches.
func MeanValues(pms []trends.PlayerMatch) map[string]float64 {
	out := map[string]float64{}
	for _, pm := range pms {
		for k, v := range Values(pm) {
			out[k] += v / float64(len(pms))
		}
	}
	return out
}

func (b *Benchmark) add(rep *analysis.Report, skip []string) {
	if rep.InPlaySeconds < minMatchSeconds {
		return
	}
	b.Replays++
	for _, p := range rep.Players {
		if slices.Contains(skip, p.Name) || p.InPlaySeconds < minPlayerSeconds {
			continue
		}
		pm, _ := trends.ForPlayer(rep, p.Name)
		for k, v := range Values(pm) {
			b.Samples[k] = append(b.Samples[k], v)
		}
		b.Players++
	}
}

// Local builds the benchmark from the other players of the n most recent
// ranked matches of the playlist: matchmaking puts them at the coached
// players' level.
func Local(replays []*replay.Replay, reports map[string]*analysis.Report, players []string, playlist string, n int) *Benchmark {
	b := &Benchmark{Source: "local", Playlist: playlist, Built: time.Now().Format("2006-01-02"), Samples: map[string][]float64{}}
	rs := slices.Clone(replays)
	sort.Slice(rs, func(i, j int) bool { return rs[i].Date.After(rs[j].Date) })
	for _, r := range rs {
		rep := reports[r.ID]
		if rep == nil || !rep.Ranked || rep.Playlist != playlist {
			continue
		}
		b.add(rep, r.Coached(players))
		if b.Replays == n {
			break
		}
	}
	return b
}

const fileName = "benchmark.json"

// Load reads a benchmark built in dir, nil if there is none.
func Load(dir string) *Benchmark {
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return nil
	}
	var b Benchmark
	if json.Unmarshal(data, &b) != nil {
		return nil
	}
	return &b
}

func round(x float64) float64 { return math.Round(x*10) / 10 }
