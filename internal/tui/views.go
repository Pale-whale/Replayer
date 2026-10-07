package tui

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"replayer/internal/analysis"
	"replayer/internal/coach"
	"replayer/internal/trends"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true)
	winStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	lossStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	blueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	orangeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	infoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

// Width of the label column of the metric tables.
const labelWidth = 24

func teamStyle(team int) lipgloss.Style {
	if team == 1 {
		return orangeStyle
	}
	return blueStyle
}

func (m Model) detailContent(it item) string {
	r := it.r
	var b strings.Builder
	us, them := it.scores()
	res := lossStyle.Render(fmt.Sprintf("Défaite %d-%d", us, them))
	if us > them {
		res = winStyle.Render(fmt.Sprintf("Victoire %d-%d", us, them))
	}
	fmt.Fprintf(&b, "\n  %s  %s\n  %s · %s %dv%d · %s\n\n", titleStyle.Render(r.Map), res,
		r.Date.Format("02/01/2006 15:04"), r.MatchType, r.TeamSize, r.TeamSize, r.ID)

	coached := r.Coached(it.players)
	for team := range 2 {
		name, score := "Bleu", r.Team0Score
		if team == 1 {
			name, score = "Orange", r.Team1Score
		}
		fmt.Fprintf(&b, "  %s\n", teamStyle(team).Render(fmt.Sprintf("%-6s %d", name, score)))
		fmt.Fprintf(&b, "  %s\n", dimStyle.Render(fmt.Sprintf("%-24s %6s %5s %5s %7s %5s", "", "Score", "Buts", "PD", "Arrêts", "Tirs")))
		for _, p := range r.Players {
			if p.Team != team {
				continue
			}
			mark := "  "
			if slices.Contains(coached, p.Name) {
				mark = "★ "
			}
			fmt.Fprintf(&b, "  %s%-22s %6d %5d %5d %7d %5d\n", mark, p.Name, p.Score, p.Goals, p.Assists, p.Saves, p.Shots)
		}
		b.WriteString("\n")
	}

	b.WriteString("  " + titleStyle.Render("Analyse") + "\n")
	switch rep, err := m.reports[r.ID], m.reportErrs[r.ID]; {
	case err != nil:
		b.WriteString("  " + errStyle.Render("analyse indisponible : "+err.Error()) + "\n\n")
	case rep == nil:
		b.WriteString("  " + infoStyle.Render("analyse en cours…") + "\n\n")
	default:
		b.WriteString(m.legend() + "\n")
		b.WriteString(metricsHeader("") + "\n")
		for _, p := range rep.Players {
			pm, _ := trends.ForPlayer(rep, p.Name)
			mark := "  "
			if slices.Contains(coached, p.Name) {
				mark = "★ "
			}
			label := teamStyle(p.Team).Render(pad(p.Name, labelWidth-2))
			b.WriteString("  " + mark + label + metricCells(func(met trends.Metric) string { return num(met.Get(pm)) }) + "\n")
		}
		b.WriteString(m.refRow(rep.Playlist))
		won := map[string]int{}
		for _, ko := range rep.Kickoffs {
			won[ko.Winner]++
		}
		fmt.Fprintf(&b, "\n  Kickoffs : %s %d · %s %d · neutres %d\n\n",
			blueStyle.Render("bleu"), won["bleu"], orangeStyle.Render("orange"), won["orange"], won["neutre"])
	}

	if rep := m.reports[r.ID]; rep != nil && len(rep.Goals) > 0 {
		b.WriteString("  " + titleStyle.Render("Buts") + "\n\n")
		for _, g := range rep.Goals {
			fmt.Fprintf(&b, "  But %s — %s, 2 s avant\n", g.Clock, teamStyle(g.ScoringTeam).Render(g.Scorer))
			b.WriteString(colorMap(g.FieldMap) + "\n")
		}
	} else if len(r.Goals) > 0 {
		b.WriteString("  " + titleStyle.Render("Buts") + "\n")
		for _, g := range r.Goals {
			s := int(g.Second)
			fmt.Fprintf(&b, "  %s  %s\n", dimStyle.Render(fmt.Sprintf("%2d:%02d", s/60, s%60)), teamStyle(g.Team).Render(g.Player))
		}
	}
	return b.String()
}

func (m Model) trendsContent() string {
	t := m.trends
	var b strings.Builder
	wins, ko := 0, [3]int{}
	for _, mt := range t.Matches {
		if mt.Won {
			wins++
		}
		ko[0] += mt.KickoffsWon
		ko[1] += mt.KickoffsLost
		ko[2] += mt.KickoffsNeutral
	}
	fmt.Fprintf(&b, "\n  %s  %s\n", titleStyle.Render(fmt.Sprintf("Tendances sur %d matchs", len(t.Matches))),
		fmt.Sprintf("%s · %s", winStyle.Render(fmt.Sprintf("%d V", wins)), lossStyle.Render(fmt.Sprintf("%d D", len(t.Matches)-wins))))
	fmt.Fprintf(&b, "  Kickoffs de l'équipe : %d gagnés · %d perdus · %d neutres\n\n", ko[0], ko[1], ko[2])
	b.WriteString(m.legend() + "\n\n")

	for _, pt := range t.Trends {
		fmt.Fprintf(&b, "  %s  %s\n", titleStyle.Render("★ "+pt.Name), dimStyle.Render(fmt.Sprintf("%d matchs, %d V", pt.Matches, pt.Wins)))
		b.WriteString(metricsHeader("Match (ancien → récent)") + "\n")
		for _, mt := range t.Matches {
			pm, ok := mt.Players[pt.Name]
			if !ok {
				continue
			}
			res, style := "D", lossStyle
			if mt.Won {
				res, style = "V", winStyle
			}
			date := mt.Date[8:10] + "/" + mt.Date[5:7]
			score := fmt.Sprintf("%s %d-%d", res, mt.ScoreUs, mt.ScoreThem)
			label := date + " " + pad(mt.Map, labelWidth-len(date)-len(score)-2) + " " + style.Render(score)
			b.WriteString("  " + label + metricCells(func(met trends.Metric) string { return num(met.Get(pm)) }) + "\n")
		}
		b.WriteString("  " + titleStyle.Render(pad("Moyenne", labelWidth)) +
			metricCells(func(met trends.Metric) string { return num(pt.Mean[met.Key]) }) + "\n")
		b.WriteString(m.refRow(coach.DominantPlaylist(m.trendReplays, m.reports)))
		if len(pt.Evolution) > 0 {
			b.WriteString("  " + pad("Évolution (réc.-anc.)", labelWidth) + metricCells(func(met trends.Metric) string {
				v := pt.Evolution[met.Key]
				if v == 0 {
					return "="
				}
				return signed(v)
			}) + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) legend() string {
	var parts []string
	for _, met := range trends.Metrics {
		parts = append(parts, met.Label+" "+met.Help)
	}
	parts = append(parts, "Réf. = médiane des joueurs de même niveau (compteurs ramenés à 5 min de jeu)")
	return dimStyle.PaddingLeft(2).Width(max(m.width-2, 20)).Render(strings.Join(parts, " · "))
}

// refRow shows the median of the level reference of the playlist.
func (m Model) refRow(playlist string) string {
	b := m.bench(playlist)
	if b == nil {
		return ""
	}
	label := fmt.Sprintf("%s (méd., n=%d)", b.Label(), b.Players)
	return "  " + infoStyle.Render(pad(label, labelWidth)) + metricCells(func(met trends.Metric) string {
		if st, ok := b.Stat(met.Key); ok {
			return num(st.Median)
		}
		return "-"
	}) + "\n"
}

// colorMap colors a field map: blue players, orange players and the ball
// on the field, each legend line in its team color.
func colorMap(lines []string) string {
	var b strings.Builder
	for i, line := range lines {
		switch {
		case i == 0:
			line = strings.Replace(strings.Replace(line, "BLEU", blueStyle.Render("BLEU"), 1), "ORANGE", orangeStyle.Render("ORANGE"), 1)
		case i < analysis.MapRows:
			var l strings.Builder
			for _, c := range line {
				switch {
				case c >= '1' && c <= '9':
					l.WriteString(blueStyle.Render(string(c)))
				case c >= 'A' && c <= 'Z':
					l.WriteString(orangeStyle.Render(string(c)))
				case c == 'o':
					l.WriteString(titleStyle.Render("o"))
				case c == ' ':
					l.WriteRune(c)
				default:
					l.WriteString(dimStyle.Render(string(c)))
				}
			}
			line = l.String()
		case len(line) > 2 && line[2] >= '1' && line[2] <= '9':
			line = blueStyle.Render(line)
		case len(line) > 2 && line[2] >= 'A' && line[2] <= 'Z':
			line = orangeStyle.Render(line)
		}
		b.WriteString("  " + line + "\n")
	}
	return b.String()
}

func metricsHeader(label string) string {
	return dimStyle.Render("  " + pad(label, labelWidth) + metricCells(func(met trends.Metric) string { return met.Label }))
}

func metricCells(cell func(trends.Metric) string) string {
	var b strings.Builder
	for _, met := range trends.Metrics {
		fmt.Fprintf(&b, " %5s", cell(met))
	}
	return b.String()
}

// num formats a metric: integers as is, small averages with one decimal.
func num(v float64) string {
	switch {
	case v == math.Trunc(v), math.Abs(v) >= 10:
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

func signed(v float64) string {
	if v > 0 {
		return "+" + num(v)
	}
	return num(v)
}

// pad truncates or pads s to n runes.
func pad(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s + strings.Repeat(" ", n-len(r))
}
