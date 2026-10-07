// Package rank handles Rocket League ranks and playlists. The rank of the
// players is not in the replays, so it comes from the config.
package rank

import (
	"fmt"
	"strings"
)

var tiers = []struct{ short, long string }{
	{"B", "bronze"}, {"S", "silver"}, {"G", "gold"}, {"P", "platinum"},
	{"D", "diamond"}, {"C", "champion"}, {"GC", "grand-champion"},
}

// Normalize turns "c3", "C3" or "champion-3" into "C3". "SSL" stays "SSL".
func Normalize(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "ssl" || s == "supersonic-legend" {
		return "SSL", nil
	}
	for _, t := range tiers {
		for _, prefix := range []string{t.long + "-", strings.ToLower(t.short)} {
			div, ok := strings.CutPrefix(s, prefix)
			if ok && len(div) == 1 && div >= "1" && div <= "3" {
				return t.short + div, nil
			}
		}
	}
	return "", fmt.Errorf("rang inconnu %q (ex : C3, D2, GC1, SSL)", s)
}

// Ballchasing returns the value of the min-rank/max-rank API filters. The
// API only documents a single "grand-champion" value above champion.
func Ballchasing(normalized string) string {
	if normalized == "SSL" || strings.HasPrefix(normalized, "GC") {
		return "grand-champion"
	}
	for _, t := range tiers {
		if div, ok := strings.CutPrefix(normalized, t.short); ok && len(div) == 1 {
			return t.long + "-" + div
		}
	}
	return ""
}

// Playlist ids replicated in the replays (ProjectX.GRI_X:ReplicatedGamePlaylist),
// named like the ballchasing API.
var playlists = map[int]string{
	1: "unranked-duels", 2: "unranked-doubles", 3: "unranked-standard", 4: "unranked-chaos",
	6: "private", 10: "ranked-duels", 11: "ranked-doubles", 12: "ranked-solo-standard",
	13: "ranked-standard", 27: "ranked-hoops", 28: "ranked-rumble", 29: "ranked-dropshot",
	30: "ranked-snowday",
}

// PlaylistName returns the name of a playlist id, "playlist-<id>" if unknown.
func PlaylistName(id int) string {
	if n, ok := playlists[id]; ok {
		return n
	}
	return fmt.Sprintf("playlist-%d", id)
}

// Ranked tells if the playlist is a competitive one.
func Ranked(id int) bool { return strings.HasPrefix(playlists[id], "ranked-") }
