package coach

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"replayer/internal/store"
)

// Editable parts of the coach prompt, in <workdir>/prompts/. They are
// created with these defaults and never overwritten. The generated part
// (session, players, level, data files, memory) is added around them.
var defaultPrompts = []struct{ file, text string }{
	{"coach.md", `Tu es un coach Rocket League expérimenté. Ton but : faire progresser les joueurs coachés.

Règles :
- Appuie chaque conseil sur des données. N'invente rien : si une info n'est pas dans les données, dis-le.
- Quand tu parles de positions, dessine un mini-terrain ASCII au format des field_map (vue de dessus, but bleu à gauche, 1 2 3 = bleus, A B C = orange, o = balle) et donne les hauteurs en % du plafond. Ne donne jamais de coordonnées brutes en uu.
- Pour creuser au-delà, tu peux extraire les frames réseau d'un replay avec "rrrocket -n <fichier.replay>". La sortie fait des dizaines de Mo : ne la lis jamais en entier, écris un script qui en tire ce dont tu as besoin. Le plafond est à z = 2044 uu.
- Réponds en français, tutoie, sois direct et concret.
`},
	{"match.md", `Structure ta première réponse : résumé du match, puis 2 ou 3 axes de progression prioritaires par joueur coaché avec les preuves (et leur place par rapport à la référence), puis des exercices concrets (free play, training, ateliers) et un objectif pour la prochaine session. Ensuite, réponds aux questions sur le match.
`},
	{"trends.md", `Cherche ce qui revient d'un match à l'autre plutôt que les accidents d'un seul match, et ce qui progresse ou régresse. Relie les tendances aux résultats (victoires/défaites, buts encaissés).
Structure ta première réponse : bilan de la série, puis les 2 ou 3 problèmes récurrents prioritaires par joueur coaché avec les chiffres qui le montrent (et leur place par rapport à la référence), ce qui s'améliore, puis un plan d'entraînement concret pour les prochaines sessions. Ensuite, réponds aux questions.
`},
}

// PromptFiles returns the editable prompt files, created with their
// default text if missing.
func PromptFiles(s store.Store) ([]string, error) {
	dir := filepath.Join(s.Dir, "prompts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var paths []string
	for _, p := range defaultPrompts {
		path := filepath.Join(dir, p.file)
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			if err := os.WriteFile(path, []byte(p.text), 0o644); err != nil {
				return nil, err
			}
		}
		paths = append(paths, path)
	}
	return paths, nil
}

// readPrompt returns the content of an editable prompt file.
func readPrompt(s store.Store, file string) (string, error) {
	if _, err := PromptFiles(s); err != nil {
		return "", err
	}
	b, err := os.ReadFile(filepath.Join(s.Dir, "prompts", file))
	return string(b), err
}
