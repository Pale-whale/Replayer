package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"replayer/internal/analysis"
	"replayer/internal/rank"
	"replayer/internal/replay"
	"replayer/internal/store"
)

// TokenURL is where a ballchasing.com account gets its API token.
const TokenURL = "https://ballchasing.com/upload"

// Ballchasing downloads replays of a rank from ballchasing.com. The free
// tier allows 1 file download per second and 200 per hour.
type Ballchasing struct {
	Token   string
	BaseURL string        // default https://ballchasing.com/api
	Wait    time.Duration // between two downloads, default 1.1 s
	Count   int           // replays to fetch, default 40
	HTTP    *http.Client
}

func (c Ballchasing) get(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	base := c.BaseURL
	if base == "" {
		base = "https://ballchasing.com/api"
	}
	u := base + path
	if query != nil {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.Token)
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: time.Minute}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		resp.Body.Close()
		return nil, fmt.Errorf("ballchasing %s: %s %s", path, resp.Status, body)
	}
	return resp, nil
}

// Build downloads the most recent replays of the playlist where every
// player has the rank (normalized, e.g. "C3"), analyses them and saves the
// benchmark in dir. Replays already downloaded are not fetched again.
func (c Ballchasing) Build(ctx context.Context, playlist, rk, dir string, progress func(string)) (*Benchmark, error) {
	count, wait := c.Count, c.Wait
	if count == 0 {
		count = 40
	}
	if wait == 0 {
		wait = 1100 * time.Millisecond
	}
	br := rank.Ballchasing(rk)
	progress("ballchasing : recherche des replays " + playlist + " " + rk + "…")
	resp, err := c.get(ctx, "/replays", url.Values{
		"playlist": {playlist}, "min-rank": {br}, "max-rank": {br},
		"count": {fmt.Sprint(count)}, "sort-by": {"replay-date"}, "sort-dir": {"desc"},
	})
	if err != nil {
		return nil, err
	}
	var list struct {
		List []struct {
			ID string `json:"id"`
		} `json:"list"`
	}
	err = json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("ballchasing: %w", err)
	}
	if len(list.List) == 0 {
		return nil, fmt.Errorf("ballchasing : aucun replay %s %s", playlist, rk)
	}

	files := filepath.Join(dir, "replays")
	if err := os.MkdirAll(files, 0o755); err != nil {
		return nil, err
	}
	downloaded := false
	for i, rp := range list.List {
		path := filepath.Join(files, rp.ID+".replay")
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if downloaded {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}
		progress(fmt.Sprintf("ballchasing : téléchargement %d/%d…", i+1, len(list.List)))
		if err := c.download(ctx, rp.ID, path); err != nil {
			return nil, err
		}
		downloaded = true
	}

	b := &Benchmark{Source: "ballchasing", Playlist: playlist, Rank: rk, Built: time.Now().Format("2006-01-02"), Samples: map[string][]float64{}}
	for i, rp := range list.List {
		progress(fmt.Sprintf("ballchasing : analyse %d/%d…", i+1, len(list.List)))
		r, err := replay.ParseFile(filepath.Join(files, rp.ID+".replay"))
		if err != nil {
			continue
		}
		rep, err := analysis.File(r, nil)
		if err != nil {
			continue
		}
		b.add(rep, nil)
	}
	if b.Players == 0 {
		return nil, fmt.Errorf("ballchasing : aucun replay analysable")
	}
	return b, store.WriteJSON(filepath.Join(dir, fileName), b)
}

func (c Ballchasing) download(ctx context.Context, id, path string) error {
	resp, err := c.get(ctx, "/replays/"+id+"/file", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	tmp := path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
