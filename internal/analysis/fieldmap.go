package analysis

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Field size of a standard (soccar) arena and mini-map size in cells.
const (
	fieldX   = 4096.0
	fieldY   = 5120.0
	ceilingZ = 2044.0
	mapCols  = 41
	mapRows  = 7
)

// MapRows is the number of lines of a field map before its legend: the
// header, the top border, the rows and the bottom border.
const MapRows = mapRows + 3

func heightPct(z float64) int {
	return int(math.Round(math.Max(0, math.Min(z/ceilingZ*100, 100))))
}

// mapLabels gives "1", "2"… to the blue players and "A", "B"… to the orange
// ones, sorted by name.
func mapLabels(cars []carSnap) map[string]string {
	sorted := append([]carSnap(nil), cars...)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		return a.Team < b.Team || a.Team == b.Team && a.Player < b.Player
	})
	labels := map[string]string{}
	n := [2]int{}
	for _, c := range sorted {
		if c.Team == 0 {
			labels[c.Player] = string(rune('1' + n[0]))
		} else {
			labels[c.Player] = string(rune('A' + n[1]))
		}
		n[c.Team]++
	}
	return labels
}

func mapCell(p vec) (row, col int) {
	col = int(math.Round((p.Y + fieldY) / (2 * fieldY) * (mapCols - 1)))
	row = int((fieldX - p.X) / (2 * fieldX) * mapRows)
	return min(max(row, 0), mapRows-1), min(max(col, 0), mapCols-1)
}

// fieldMap draws the field seen from above, blue goal on the left, with the
// players (labels) and the ball (o), followed by a legend with boost and
// height in % of the ceiling.
func fieldMap(s snapshot, labels map[string]string) []string {
	third := func(y float64) int { return int(math.Round((y + fieldY) / (2 * fieldY) * (mapCols - 1))) }
	marks := map[int]rune{third(-thirdY): ':', third(0): '|', third(thirdY): ':'}

	grid := make([][]rune, mapRows)
	for r := range grid {
		grid[r] = []rune(strings.Repeat(" ", mapCols))
		for c, m := range marks {
			grid[r][c] = m
		}
	}
	occupied := map[[2]int]bool{}
	place := func(p vec, label rune) {
		r, c := mapCell(p)
		for _, dc := range []int{0, 1, -1, 2, -2} {
			if cc := c + dc; cc >= 0 && cc < mapCols && !occupied[[2]int{r, cc}] {
				grid[r][cc], occupied[[2]int{r, cc}] = label, true
				return
			}
		}
	}
	cars := append([]carSnap(nil), s.Cars...)
	sort.Slice(cars, func(i, j int) bool { return labels[cars[i].Player] < labels[cars[j].Player] })
	r, c := mapCell(s.Ball) // the ball keeps its exact cell
	grid[r][c], occupied[[2]int{r, c}] = 'o', true
	for _, car := range cars {
		place(car.Pos, []rune(labels[car.Player])[0])
	}

	border := []rune(strings.Repeat("-", mapCols))
	for c, m := range marks {
		border[c] = m
	}
	lines := []string{
		"   BLEU" + strings.Repeat(" ", mapCols-9) + "ORANGE",
		"  +" + string(border) + "+",
	}
	goalRow := mapRows / 2
	for r, row := range grid {
		left, right := " ", " "
		if r == goalRow {
			left, right = "[", "]"
		}
		lines = append(lines, " "+left+"|"+string(row)+"|"+right)
	}
	lines = append(lines, "  +"+string(border)+"+")

	for _, car := range cars {
		lines = append(lines, fmt.Sprintf("  %s %-16s boost %3.0f  hauteur %3d%%", labels[car.Player], truncate(car.Player, 16), car.Boost, heightPct(car.Pos.Z)))
	}
	lines = append(lines, fmt.Sprintf("  o %-16s            hauteur %3d%%", "balle", heightPct(s.Ball.Z)))
	return lines
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
