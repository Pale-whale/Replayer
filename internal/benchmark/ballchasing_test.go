package benchmark

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestBallchasingBuild(t *testing.T) {
	// A real replay makes the analysis succeed; junk only tests the API side.
	content := []byte("not a replay")
	if dir := os.Getenv("RL_DEMOS"); dir != "" {
		b, err := os.ReadFile(filepath.Join(dir, "E31BC47211F1C1D34A3BDABA21291C79.replay"))
		if err == nil {
			content = b
		}
	}

	var mu sync.Mutex
	var downloads []time.Time
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/replays":
			q := r.URL.Query()
			if q.Get("playlist") != "ranked-doubles" || q.Get("min-rank") != "champion-3" || q.Get("max-rank") != "champion-3" {
				t.Errorf("query %v", q)
			}
			fmt.Fprint(w, `{"list":[{"id":"a"},{"id":"b"},{"id":"c"}]}`)
		default:
			mu.Lock()
			downloads = append(downloads, time.Now())
			mu.Unlock()
			w.Write(content)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	c := Ballchasing{Token: "secret", BaseURL: srv.URL, Wait: 50 * time.Millisecond}
	b, err := c.Build(context.Background(), "ranked-doubles", "C3", dir, func(string) {})
	if len(downloads) != 3 {
		t.Fatalf("%d downloads", len(downloads))
	}
	for i := 1; i < len(downloads); i++ {
		if downloads[i].Sub(downloads[i-1]) < 50*time.Millisecond {
			t.Error("downloads not spaced")
		}
	}
	if string(content) == "not a replay" {
		if err == nil {
			t.Error("junk replays should not make a benchmark")
		}
	} else {
		if err != nil || b.Players != 12 || Load(dir) == nil {
			t.Fatalf("benchmark: %+v, %v", b, err)
		}
	}

	// Resume: files already there are not downloaded again.
	c.Build(context.Background(), "ranked-doubles", "C3", dir, func(string) {})
	if len(downloads) != 3 {
		t.Errorf("%d downloads after resume", len(downloads))
	}

	if _, err := (Ballchasing{Token: "wrong", BaseURL: srv.URL}).Build(context.Background(), "ranked-doubles", "C3", t.TempDir(), func(string) {}); err == nil {
		t.Error("bad token should fail")
	}
}
