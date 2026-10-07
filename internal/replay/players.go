package replay

import "slices"

// ApplyAliases renames the players known under another name (alias -> main
// name) in the scoreboard, the goals and the recorder.
func (r *Replay) ApplyAliases(aliases map[string]string) {
	rename := func(s *string) {
		if n, ok := aliases[*s]; ok {
			*s = n
		}
	}
	for i := range r.Players {
		rename(&r.Players[i].Name)
	}
	for i := range r.Goals {
		rename(&r.Goals[i].Player)
	}
	rename(&r.Recorder)
}

// Coached returns the configured players present in r, or the recorder if
// none is configured or present.
func (r *Replay) Coached(players []string) []string {
	var out []string
	for _, p := range r.Players {
		if slices.Contains(players, p.Name) {
			out = append(out, p.Name)
		}
	}
	if len(out) == 0 && r.Recorder != "" {
		out = []string{r.Recorder}
	}
	return out
}

// OurTeam is the team of the coached players.
func (r *Replay) OurTeam(players []string) int {
	coached := r.Coached(players)
	for _, p := range r.Players {
		if slices.Contains(coached, p.Name) {
			return p.Team
		}
	}
	return r.RecorderTeam
}

// Scores returns the score from the coached players' point of view.
func (r *Replay) Scores(players []string) (us, them int) {
	if r.OurTeam(players) == 0 {
		return r.Team0Score, r.Team1Score
	}
	return r.Team1Score, r.Team0Score
}
