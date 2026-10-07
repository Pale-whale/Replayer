package rank

import "testing"

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"C3": "C3", "c3": "C3", "champion-3": "C3", " d2 ": "D2", "gc1": "GC1",
		"grand-champion-2": "GC2", "SSL": "SSL", "b1": "B1", "platinum-2": "P2",
	} {
		if got, err := Normalize(in); err != nil || got != want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "c4", "x1", "champion"} {
		if _, err := Normalize(in); err == nil {
			t.Errorf("Normalize(%q) should fail", in)
		}
	}
}

func TestBallchasing(t *testing.T) {
	for in, want := range map[string]string{"C3": "champion-3", "D1": "diamond-1", "GC2": "grand-champion", "SSL": "grand-champion"} {
		if got := Ballchasing(in); got != want {
			t.Errorf("Ballchasing(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestPlaylist(t *testing.T) {
	if PlaylistName(11) != "ranked-doubles" || !Ranked(11) || Ranked(2) || PlaylistName(99) != "playlist-99" {
		t.Error("playlist mapping")
	}
}
