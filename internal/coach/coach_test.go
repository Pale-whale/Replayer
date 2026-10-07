package coach

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"replayer/internal/analysis"
	"replayer/internal/benchmark"
	"replayer/internal/replay"
	"replayer/internal/store"
)

func TestCommandUsesEditedPrompts(t *testing.T) {
	s := store.Store{Dir: t.TempDir()}
	files, err := PromptFiles(s)
	if err != nil || len(files) != 3 {
		t.Fatalf("prompt files: %v %v", files, err)
	}
	if err := os.WriteFile(files[0], []byte("PERSONA PERSO"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PromptFiles(s); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(files[0]); string(b) != "PERSONA PERSO" {
		t.Fatal("an existing prompt file must not be overwritten")
	}

	r := &replay.Replay{ID: "abc", Path: filepath.Join(s.Dir, "x.replay"), Recorder: "Moi", Players: []replay.Player{{Name: "Moi"}}}
	env := Env{Store: s, Players: []string{"Moi"}, Rank: func(string) string { return "C3" },
		Replays: []*replay.Replay{r}, Reports: map[string]*analysis.Report{},
		Bench: func(string) *benchmark.Benchmark { return nil }}
	cmd, err := Command(env, r)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.Index(cmd.Args, "--append-system-prompt")
	if i < 0 {
		t.Fatalf("args %v", cmd.Args)
	}
	prompt := cmd.Args[i+1]
	for _, want := range []string{"PERSONA PERSO", "Joueurs coachés : Moi", "match.json", "Structure ta première réponse : résumé du match", "journal.md"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt misses %q:\n%s", want, prompt)
		}
	}
	if !strings.HasPrefix(prompt, "PERSONA PERSO") {
		t.Error("coach.md must come first")
	}
}
