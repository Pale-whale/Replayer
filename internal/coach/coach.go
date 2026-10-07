// Package coach prepares a working directory for a replay (or a set of
// replays) and builds the interactive `claude` command that coaches the
// players on it.
package coach

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"replayer/internal/analysis"
	"replayer/internal/benchmark"
	"replayer/internal/replay"
	"replayer/internal/store"
	"replayer/internal/trends"
)

// historyLimit is the number of past matches per player in history.json.
const historyLimit = 50

// Env is what a coaching session needs besides the match itself.
type Env struct {
	Store   store.Store
	Players []string
	// Rank is the configured rank for a playlist ("" if none).
	Rank func(playlist string) string
	// Every replay known and their analyses, for the history.
	Replays []*replay.Replay
	Reports map[string]*analysis.Report
	// Bench returns the reference of a playlist, nil if none.
	Bench func(playlist string) *benchmark.Benchmark
}

// Command writes the session files for r and returns the claude command to
// run in that session directory. Without analysis of r, the session works
// on the header stats only.
func Command(env Env, r *replay.Replay) (*exec.Cmd, error) {
	dir, err := env.Store.SessionDir(r.ID)
	if err != nil {
		return nil, err
	}
	if err := store.WriteJSON(filepath.Join(dir, "match.json"), r); err != nil {
		return nil, err
	}
	link := filepath.Join(dir, "replay.replay")
	os.Remove(link)
	if err := os.Symlink(r.Path, link); err != nil {
		return nil, err
	}
	for _, f := range []string{"analysis.json", "history.json", "benchmark.json"} {
		os.Remove(filepath.Join(dir, f))
	}

	coached := r.Coached(env.Players)
	rep := env.Reports[r.ID]
	data := "- analysis.json : absent, l'analyse des frames a échoué. Tu n'as que les stats de fin de match."
	level := ""
	if rep != nil {
		if err := store.WriteJSON(filepath.Join(dir, "analysis.json"), rep); err != nil {
			return nil, err
		}
		data = "- analysis.json : " + analysisDoc + "\n  Compare aussi les joueurs coachés aux autres joueurs du match."
		values := map[string]map[string]float64{}
		for _, name := range coached {
			if pm, ok := trends.ForPlayer(rep, name); ok {
				values[name] = benchmark.Values(pm)
			}
		}
		extra, err := writeContext(env, dir, rep.Playlist, values)
		if err != nil {
			return nil, err
		}
		data += extra
		level = levelInfo(env, rep.Playlist)
	}
	files := fmt.Sprintf(`- match.json : en-tête du replay (map, mode, score, scoreboard par joueur, buts). "second" d'un but = secondes depuis le début de l'enregistrement, pas l'horloge du match.
%s
- replay.replay : le fichier replay brut.`, data)
	return env.claude(dir, "match.md", session(coached, "un match", level, files), "Analyse ce match et coache-moi.")
}

// TrendsCommand writes the trends of several matches and returns the claude
// command to coach on them. The directory only depends on the team
// composition, so Claude Code asks to trust it once.
func TrendsCommand(env Env, t *trends.Trends, replays []*replay.Replay) (*exec.Cmd, error) {
	size := ""
	for _, r := range replays {
		s := fmt.Sprintf("%dv%d", r.TeamSize, r.TeamSize)
		if size == "" {
			size = s
		} else if s != size {
			size = "mixte"
		}
	}
	name := "trends-" + size + "-" + regexp.MustCompile(`[^A-Za-z0-9_-]+`).ReplaceAllString(strings.Join(t.Players, "+"), "_")
	dir, err := env.Store.SessionDir(name)
	if err != nil {
		return nil, err
	}
	matches := filepath.Join(dir, "matches")
	if err := os.RemoveAll(matches); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(matches, 0o755); err != nil {
		return nil, err
	}
	for _, f := range []string{"history.json", "benchmark.json"} {
		os.Remove(filepath.Join(dir, f))
	}
	for _, r := range replays {
		rep := env.Reports[r.ID]
		if rep == nil {
			continue
		}
		file := filepath.Join(matches, r.Date.Format("2006-01-02_1504")+"_"+r.ID+".json")
		if err := store.WriteJSON(file, map[string]any{"replay_file": r.Path, "match": r, "analysis": rep}); err != nil {
			return nil, err
		}
	}
	if err := store.WriteJSON(filepath.Join(dir, "trends.json"), t); err != nil {
		return nil, err
	}

	playlist := DominantPlaylist(replays, env.Reports)
	values := map[string]map[string]float64{}
	for _, name := range t.Players {
		var pms []trends.PlayerMatch
		for _, m := range t.Matches {
			if pm, ok := m.Players[name]; ok {
				pms = append(pms, pm)
			}
		}
		values[name] = benchmark.MeanValues(pms)
	}
	extra, err := writeContext(env, dir, playlist, values)
	if err != nil {
		return nil, err
	}

	files := fmt.Sprintf(`- trends.json : pour chaque match (du plus ancien au plus récent) le résultat, les kickoffs de l'équipe et les métriques de chaque joueur coaché ; par joueur, les moyennes et l'évolution (moitié récente - moitié ancienne). Lis d'abord "notes".
- matches/<date>_<id>.json : pour chaque match, "replay_file" (chemin du replay brut), "match" (en-tête : scoreboard, buts) et "analysis" (%s).%s`, analysisDoc, extra)
	what := fmt.Sprintf("une série de %d matchs", len(t.Matches))
	return env.claude(dir, "trends.md", session(t.Players, what, levelInfo(env, playlist), files), "Analyse mes tendances sur ces matchs et coache-moi.")
}

// DominantPlaylist is the most frequent playlist among the analysed replays.
func DominantPlaylist(replays []*replay.Replay, reports map[string]*analysis.Report) string {
	count := map[string]int{}
	best := ""
	for _, r := range replays {
		if rep := reports[r.ID]; rep != nil {
			count[rep.Playlist]++
			if count[rep.Playlist] > count[best] {
				best = rep.Playlist
			}
		}
	}
	return best
}

// writeContext writes history.json and benchmark.json for the coached
// players and returns their description for the prompt.
func writeContext(env Env, dir, playlist string, values map[string]map[string]float64) (string, error) {
	h := trends.BuildHistory(env.Replays, env.Reports, env.Players, playlist, historyLimit)
	if err := store.WriteJSON(filepath.Join(dir, "history.json"), h); err != nil {
		return "", err
	}
	doc := "\n- history.json : l'historique des joueurs coachés sur leurs " + fmt.Sprint(historyLimit) + " derniers matchs " + playlist + " (métriques par match, moyenne des 10 derniers vs des 10 d'avant). Sers-t'en pour juger la progression dans le temps."

	b := env.Bench(playlist)
	if b == nil || b.Players == 0 {
		return doc + "\n- Pas de référence de niveau disponible pour cette playlist.", nil
	}
	ref := map[string]benchmark.Stat{}
	for k := range b.Samples {
		ref[k], _ = b.Stat(k)
	}
	coached := map[string]map[string]benchmark.Comparison{}
	for name, v := range values {
		coached[name] = b.Compare(v)
	}
	source := fmt.Sprintf("les autres joueurs des %d derniers matchs classés %s des joueurs coachés (le matchmaking les place au même niveau)", b.Replays, playlist)
	if b.Source == "ballchasing" {
		source = fmt.Sprintf("%d replays %s %s téléchargés depuis ballchasing.com le %s", b.Replays, playlist, b.Rank, b.Built)
	}
	err := store.WriteJSON(filepath.Join(dir, "benchmark.json"), map[string]any{
		"notes": []string{
			"Référence : " + source + ", analysés avec les mêmes définitions qu'analysis.json. " + fmt.Sprint(b.Players) + " joueurs.",
			"reference : quartiles par métrique. coached : pour chaque joueur coaché, sa valeur et son percentile dans la référence (50 = médiane).",
			"Les compteurs (touches, kickoffs, pads, démos, 50/50…) sont ramenés à 5 min de jeu, pour la référence comme pour les joueurs coachés.",
			"Un percentile n'est pas une note : plus de temps en tiers défensif ou plus de 50/50 n'est pas forcément mieux. Interprète selon le contexte.",
		},
		"source": b.Source, "playlist": b.Playlist, "rank": b.Rank, "replays": b.Replays, "players": b.Players,
		"reference": ref, "coached": coached,
	})
	return doc + "\n- benchmark.json : la référence de niveau (" + source + ") et la place de chaque joueur coaché (percentiles). Lis ses notes.", err
}

func levelInfo(env Env, playlist string) string {
	if r := env.Rank(playlist); r != "" {
		return fmt.Sprintf("Niveau déclaré des joueurs coachés en %s : %s (le rang n'est pas dans les replays, il vient de la config). Adapte tes conseils à ce niveau.", playlist, r)
	}
	return ""
}

// claude builds the coach command: the editable coach.md, the generated
// session description, the editable instructions of the session kind
// (match.md or trends.md) and the memory instructions.
func (env Env) claude(dir, kindPrompt, sessionPrompt, firstMessage string) (*exec.Cmd, error) {
	mem, err := env.Store.MemoryDir()
	if err != nil {
		return nil, err
	}
	coachText, err := readPrompt(env.Store, "coach.md")
	if err != nil {
		return nil, err
	}
	kindText, err := readPrompt(env.Store, kindPrompt)
	if err != nil {
		return nil, err
	}
	systemPrompt := strings.TrimSpace(coachText) + "\n\n" + sessionPrompt + "\n\n" + strings.TrimSpace(kindText)
	systemPrompt += fmt.Sprintf(`

Mémoire (dossier %s, partagée entre toutes les sessions de coaching) :
- Au début, lis objectifs.md et journal.md s'ils existent. Dis si les objectifs précédents sont atteints, chiffres de history.json à l'appui.
- Après ta première réponse, ajoute à journal.md une entrée datée (## %s — le match ou la série) avec les constats chiffrés et les conseils donnés, puis réécris objectifs.md avec 2 ou 3 objectifs mesurables (métrique, valeur actuelle, cible). Mets-les à jour si la discussion fait évoluer les conseils.`,
		mem, time.Now().Format("2006-01-02"))
	cmd := exec.Command("claude", "--add-dir", mem, "--permission-mode", "acceptEdits",
		"--append-system-prompt", systemPrompt, firstMessage)
	cmd.Dir = dir
	return cmd, nil
}

const analysisDoc = `métriques calculées depuis les frames réseau, pour tous les joueurs (lis d'abord "notes" : unités, méthode, limites).
  Par joueur : répartition par tiers du terrain, temps derrière la balle, distance à la balle, rôle 1er homme / dernier homme (vs coéquipiers), vitesse et supersonique, sol/air, boost (moyenne, temps à 0, <25, >80, gros et petits pads, boost moyen au moment de prendre un gros pad), touches par tiers, possession (touches suivies par soi-même / un coéquipier, rendues à l'adversaire, 50/50 et leur issue), démos, kickoffs (allés, gagnés, perdus).
  Par but : horloge, 4 dernières touches, position/boost/vitesse/hauteur de chaque joueur 2 s avant le but et le mini-terrain ASCII correspondant (field_map). Par kickoff : position de départ de chacun, qui y va, temps jusqu'à la balle, équipe gagnante. Liste des démos avec l'horloge.`

// session describes the session: what is analysed, for whom, and the data.
func session(coached []string, what, level, files string) string {
	s := fmt.Sprintf(`Session : tu analyses %s. Joueurs coachés : %s.
Le premier nom est le joueur principal, les autres sont ses coéquipiers. team 0 = bleu, team 1 = orange.`, what, strings.Join(coached, ", "))
	if level != "" {
		s += "\n" + level
	}
	return s + "\n\nDonnées dans le dossier courant :\n" + files
}
