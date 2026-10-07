// Package coach prepares a working directory for a replay (or a set of
// replays) and builds the interactive `claude` command that coaches the
// players on it.
package coach

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"replayer/internal/analysis"
	"replayer/internal/replay"
	"replayer/internal/trends"
)

// Command writes the session files for r and returns the claude command to
// run in that session directory. rep is the frame analysis of r, nil if it
// failed: the session then works on the header stats only.
func Command(r *replay.Replay, rep *analysis.Report, players []string) (*exec.Cmd, error) {
	dir, err := sessionDir(r.ID)
	if err != nil {
		return nil, err
	}
	if err := writeJSON(filepath.Join(dir, "match.json"), r); err != nil {
		return nil, err
	}
	link := filepath.Join(dir, "replay.replay")
	os.Remove(link)
	if err := os.Symlink(r.Path, link); err != nil {
		return nil, err
	}
	analysisPath := filepath.Join(dir, "analysis.json")
	os.Remove(analysisPath)
	if rep != nil {
		if err := writeJSON(analysisPath, rep); err != nil {
			return nil, err
		}
	}
	return claude(dir, matchPrompt(r.Coached(players), rep != nil), "Analyse ce match et coache-moi."), nil
}

// TrendsCommand writes the trends of several matches and returns the claude
// command to coach on them. The directory only depends on the team
// composition, so Claude Code asks to trust it once.
func TrendsCommand(t *trends.Trends, replays []*replay.Replay, reports map[string]*analysis.Report) (*exec.Cmd, error) {
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
	dir, err := sessionDir(name)
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
	for _, r := range replays {
		rep := reports[r.ID]
		if rep == nil {
			continue
		}
		file := filepath.Join(matches, r.Date.Format("2006-01-02_1504")+"_"+r.ID+".json")
		if err := writeJSON(file, map[string]any{"replay_file": r.Path, "match": r, "analysis": rep}); err != nil {
			return nil, err
		}
	}
	if err := writeJSON(filepath.Join(dir, "trends.json"), t); err != nil {
		return nil, err
	}
	return claude(dir, trendsPrompt(t.Players, len(t.Matches)), "Analyse mes tendances sur ces matchs et coache-moi."), nil
}

func sessionDir(name string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "replayer", "sessions", name)
	return dir, os.MkdirAll(dir, 0o755)
}

func claude(dir, systemPrompt, firstMessage string) *exec.Cmd {
	cmd := exec.Command("claude", "--append-system-prompt", systemPrompt, firstMessage)
	cmd.Dir = dir
	return cmd
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

const analysisDoc = `métriques calculées depuis les frames réseau, pour tous les joueurs (lis d'abord "notes" : unités, méthode, limites).
  Par joueur : répartition par tiers du terrain, temps derrière la balle, distance à la balle, rôle 1er homme / dernier homme (vs coéquipiers), vitesse et supersonique, sol/air, boost (moyenne, temps à 0, <25, >80, gros et petits pads, boost moyen au moment de prendre un gros pad), touches par tiers, démos, kickoffs (allés, gagnés, perdus).
  Par but : horloge, 4 dernières touches, et position/boost/vitesse de chaque joueur 2 s avant le but. Par kickoff : position de départ de chacun, qui y va, temps jusqu'à la balle, équipe gagnante. Liste des démos avec l'horloge.`

const rules = `Règles :
- Appuie chaque conseil sur des données. N'invente rien : si une info n'est pas dans les données, dis-le.
- Pour creuser au-delà, tu peux extraire les frames réseau d'un replay avec "rrrocket -n <fichier.replay>". La sortie fait des dizaines de Mo : ne la lis jamais en entier, écris un script qui en tire ce dont tu as besoin.
- Réponds en français, tutoie, sois direct et concret.`

func intro(coached []string, what string) string {
	return fmt.Sprintf(`Tu es un coach Rocket League expérimenté. Tu analyses %s pour aider à progresser : %s.
Le premier nom est le joueur principal, les autres sont ses coéquipiers. team 0 = bleu, team 1 = orange.`, what, strings.Join(coached, ", "))
}

func matchPrompt(coached []string, hasAnalysis bool) string {
	data := "- analysis.json : absent, l'analyse des frames a échoué. Tu n'as que les stats de fin de match."
	if hasAnalysis {
		data = "- analysis.json : " + analysisDoc + "\n  Compare les joueurs coachés aux autres joueurs du match pour situer les chiffres."
	}
	return fmt.Sprintf(`%s

Données dans le dossier courant :
- match.json : en-tête du replay (map, mode, score, scoreboard par joueur, buts). "second" d'un but = secondes depuis le début de l'enregistrement, pas l'horloge du match.
%s
- replay.replay : le fichier replay brut.

%s
- Structure ta première réponse : résumé du match, puis 2 ou 3 axes de progression prioritaires par joueur coaché avec les preuves, puis des exercices concrets (free play, training, ateliers) et un objectif pour la prochaine session. Ensuite, réponds aux questions sur le match.`,
		intro(coached, "un match"), data, rules)
}

func trendsPrompt(coached []string, matches int) string {
	return fmt.Sprintf(`%s

Données dans le dossier courant :
- trends.json : pour chaque match (du plus ancien au plus récent) le résultat, les kickoffs de l'équipe et les métriques de chaque joueur coaché ; par joueur, les moyennes et l'évolution (moitié récente - moitié ancienne). Lis d'abord "notes".
- matches/<date>_<id>.json : pour chaque match, "replay_file" (chemin du replay brut), "match" (en-tête : scoreboard, buts) et "analysis" (%s).

%s
- Cherche ce qui revient d'un match à l'autre plutôt que les accidents d'un seul match, et ce qui progresse ou régresse. Relie les tendances aux résultats (victoires/défaites, buts encaissés).
- Structure ta première réponse : bilan de la série, puis les 2 ou 3 problèmes récurrents prioritaires par joueur coaché avec les chiffres qui le montrent, ce qui s'améliore, puis un plan d'entraînement concret pour les prochaines sessions. Ensuite, réponds aux questions.`,
		intro(coached, fmt.Sprintf("une série de %d matchs", matches)), analysisDoc, rules)
}
