// Package replay parses the header section of Rocket League .replay files.
//
// The header holds the match metadata (date, map, score, scoreboard, goals)
// and is small, so it can be read for every replay when building the list.
// Network frames (positions, boost, ...) are not handled here.
package replay

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
)

type Player struct {
	Name     string `json:"name"`
	Platform string `json:"platform,omitempty"`
	Team     int    `json:"team"`
	Score    int    `json:"score"`
	Goals    int    `json:"goals"`
	Assists  int    `json:"assists"`
	Saves    int    `json:"saves"`
	Shots    int    `json:"shots"`
	Bot      bool   `json:"bot,omitempty"`
}

type Goal struct {
	Frame  int     `json:"frame"`
	Second float64 `json:"second"`
	Player string  `json:"player"`
	Team   int     `json:"team"`
}

type Replay struct {
	Path         string    `json:"-"`
	ID           string    `json:"id"`
	Name         string    `json:"name,omitempty"`
	Date         time.Time `json:"date"`
	Map          string    `json:"map"`
	MatchType    string    `json:"match_type"`
	TeamSize     int       `json:"team_size"`
	Team0Score   int       `json:"team0_score"`
	Team1Score   int       `json:"team1_score"`
	Recorder     string    `json:"recorder"`
	RecorderTeam int       `json:"recorder_team"`
	NumFrames    int       `json:"num_frames"`
	FPS          float64   `json:"record_fps"`
	Players      []Player  `json:"players"`
	Goals        []Goal    `json:"goals"`
}

// ParseFile reads only the header part of the replay file.
func ParseFile(path string) (*Replay, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var size int32
	if err := binary.Read(f, binary.LittleEndian, &size); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if size <= 0 || size > 16<<20 {
		return nil, fmt.Errorf("%s: invalid header size %d", path, size)
	}
	buf := make([]byte, 4+int(size)) // crc + header
	if _, err := f.ReadAt(buf, 4); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	props, err := parseHeader(buf[4:])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	r := fromProps(props)
	r.Path = path
	if r.ID == "" {
		r.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if r.Date.IsZero() {
		if st, err := f.Stat(); err == nil {
			r.Date = st.ModTime()
		}
	}
	return r, nil
}

func parseHeader(b []byte) (map[string]any, error) {
	r := &reader{b: b}
	major := r.i32()
	minor := r.i32()
	if major > 865 && minor > 17 {
		r.i32() // net version
	}
	r.str() // game type, e.g. "TAGame.Replay_Soccar_TA"
	props := r.props()
	return props, r.err
}

type reader struct {
	b   []byte
	off int
	err error
}

func (r *reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || r.off+n > len(r.b) {
		r.err = fmt.Errorf("unexpected end of header at offset %d", r.off)
		return nil
	}
	s := r.b[r.off : r.off+n]
	r.off += n
	return s
}

func (r *reader) i32() int32 {
	b := r.take(4)
	if b == nil {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(b))
}

func (r *reader) u64() uint64 {
	b := r.take(8)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint64(b)
}

func (r *reader) f32() float32 {
	return math.Float32frombits(uint32(r.i32()))
}

// str reads an Unreal string: a length prefix, then either latin1/utf8 bytes
// (positive length) or UTF-16LE code units (negative length), null terminated.
func (r *reader) str() string {
	n := int(r.i32())
	switch {
	case n == 0:
		return ""
	case n < 0:
		b := r.take(-n * 2)
		u := make([]uint16, len(b)/2)
		for i := range u {
			u[i] = binary.LittleEndian.Uint16(b[i*2:])
		}
		return strings.TrimRight(string(utf16.Decode(u)), "\x00")
	default:
		return strings.TrimRight(string(r.take(n)), "\x00")
	}
}

func (r *reader) props() map[string]any {
	m := map[string]any{}
	for r.err == nil {
		name := r.str()
		if name == "None" || r.err != nil {
			break
		}
		typ := r.str()
		size := r.i32()
		r.i32() // array index
		switch typ {
		case "IntProperty":
			m[name] = int(r.i32())
		case "StrProperty", "NameProperty":
			m[name] = r.str()
		case "FloatProperty":
			m[name] = float64(r.f32())
		case "BoolProperty":
			b := r.take(1)
			m[name] = b != nil && b[0] != 0
		case "QWordProperty":
			m[name] = r.u64()
		case "ByteProperty":
			kind := r.str()
			switch kind {
			case "None": // raw byte value, e.g. "Term" in RewardData
				r.take(int(size))
				m[name] = kind
			case "OnlinePlatform_Steam", "OnlinePlatform_PS4":
				m[name] = kind
			default:
				m[name] = r.str()
			}
		case "ArrayProperty":
			n := int(r.i32())
			arr := make([]map[string]any, 0, max(n, 0))
			for i := 0; i < n && r.err == nil; i++ {
				arr = append(arr, r.props())
			}
			m[name] = arr
		case "StructProperty":
			r.str() // struct name
			m[name] = r.props()
		default:
			r.err = fmt.Errorf("unknown property type %q for %q", typ, name)
		}
	}
	return m
}

func fromProps(p map[string]any) *Replay {
	r := &Replay{
		ID:           getStr(p, "Id"),
		Name:         getStr(p, "ReplayName"),
		Map:          getStr(p, "MapName"),
		MatchType:    getStr(p, "MatchType"),
		TeamSize:     getInt(p, "TeamSize"),
		Team0Score:   getInt(p, "Team0Score"),
		Team1Score:   getInt(p, "Team1Score"),
		Recorder:     getStr(p, "PlayerName"),
		RecorderTeam: getInt(p, "PrimaryPlayerTeam"),
		NumFrames:    getInt(p, "NumFrames"),
		FPS:          getFloat(p, "RecordFPS"),
	}
	for _, layout := range []string{"2006-01-02 15-04-05", "2006-01-02:15-04"} {
		if t, err := time.ParseInLocation(layout, getStr(p, "Date"), time.Local); err == nil {
			r.Date = t
			break
		}
	}
	for _, ps := range getArr(p, "PlayerStats") {
		r.Players = append(r.Players, Player{
			Name:     getStr(ps, "Name"),
			Platform: strings.TrimPrefix(getStr(ps, "Platform"), "OnlinePlatform_"),
			Team:     getInt(ps, "Team"),
			Score:    getInt(ps, "Score"),
			Goals:    getInt(ps, "Goals"),
			Assists:  getInt(ps, "Assists"),
			Saves:    getInt(ps, "Saves"),
			Shots:    getInt(ps, "Shots"),
			Bot:      getBool(ps, "bBot"),
		})
	}
	for _, g := range getArr(p, "Goals") {
		goal := Goal{Frame: getInt(g, "frame"), Player: getStr(g, "PlayerName"), Team: getInt(g, "PlayerTeam")}
		if r.FPS > 0 {
			goal.Second = float64(goal.Frame) / r.FPS
		}
		r.Goals = append(r.Goals, goal)
	}
	return r
}

func getStr(p map[string]any, k string) string    { v, _ := p[k].(string); return v }
func getInt(p map[string]any, k string) int       { v, _ := p[k].(int); return v }
func getFloat(p map[string]any, k string) float64 { v, _ := p[k].(float64); return v }
func getBool(p map[string]any, k string) bool     { v, _ := p[k].(bool); return v }
func getArr(p map[string]any, k string) []map[string]any {
	v, _ := p[k].([]map[string]any)
	return v
}
