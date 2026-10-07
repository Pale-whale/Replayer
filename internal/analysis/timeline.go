// Package analysis turns the network frames of a replay (decoded by the
// rrrocket CLI) into compact coaching metrics.
package analysis

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Subset of `rrrocket -n` JSON output that we use.
type netReplay struct {
	// aliases renames players (alias -> main name), set by File.
	aliases       map[string]string
	Objects       []string `json:"objects"`
	Names         []string `json:"names"`
	NetworkFrames struct {
		Frames []netFrame `json:"frames"`
	} `json:"network_frames"`
}

type netFrame struct {
	Time      float64 `json:"time"`
	Delta     float64 `json:"delta"`
	NewActors []struct {
		ActorID  int `json:"actor_id"`
		ObjectID int `json:"object_id"`
	} `json:"new_actors"`
	DeletedActors []int `json:"deleted_actors"`
	UpdatedActors []struct {
		ActorID   int     `json:"actor_id"`
		ObjectID  int     `json:"object_id"`
		Attribute netAttr `json:"attribute"`
	} `json:"updated_actors"`
}

type vec struct{ X, Y, Z float64 }

type actorRef struct {
	Active bool `json:"active"`
	Actor  int  `json:"actor"`
}

type demolish struct {
	Attacker actorRef `json:"attacker"`
	Victim   actorRef `json:"victim"`
}

// Format of the older Demolish and DemolishFx attributes.
type demolishFlat struct {
	AttackerFlag bool `json:"attacker_flag"`
	Attacker     int  `json:"attacker"`
	VictimFlag   bool `json:"victim_flag"`
	Victim       int  `json:"victim"`
}

func (d *demolishFlat) demolish() *demolish {
	if d == nil {
		return nil
	}
	return &demolish{Attacker: actorRef{d.AttackerFlag, d.Attacker}, Victim: actorRef{d.VictimFlag, d.Victim}}
}

type netAttr struct {
	RigidBody *struct {
		Location       vec  `json:"location"`
		LinearVelocity *vec `json:"linear_velocity"`
	} `json:"RigidBody"`
	ActiveActor     *actorRef `json:"ActiveActor"`
	ReplicatedBoost *struct {
		BoostAmount int `json:"boost_amount"`
	} `json:"ReplicatedBoost"`
	Int     *int    `json:"Int"`
	Byte    *int    `json:"Byte"`
	Boolean *bool   `json:"Boolean"`
	String  *string `json:"String"`
	// The demolition attribute changed name across game versions.
	Demolish         *demolishFlat `json:"Demolish"`
	DemolishFx       *demolishFlat `json:"DemolishFx"`
	DemolishExtended *demolish     `json:"DemolishExtended"`
}

// Boost is replicated as 0-255 and drains at 1/3 of the tank per second.
const (
	boostMax       = 255.0
	boostDrainRate = boostMax / 3
)

type carSnap struct {
	Player string
	Team   int
	Pos    vec
	Vel    vec
	Boost  float64 // 0-100
}

// snapshot is the game state at one frame where the ball is in play.
type snapshot struct {
	Frame    int
	Time     float64
	Delta    float64
	Clock    int // seconds remaining (elapsed in overtime: the counter goes up)
	Overtime bool
	Ball     vec
	BallVel  vec
	Cars     []carSnap
}

type pickup struct {
	Time   float64
	Player string
	Big    bool
	Before float64 // boost 0-100 before pickup
}

type demo struct {
	Time     float64
	Clock    int
	Overtime bool
	Attacker string
	Victim   string
}

type touch struct {
	Frame  int
	Time   float64
	Player string
}

type timeline struct {
	Snaps   []snapshot
	Pickups []pickup
	Demos   []demo
	// Touches from the PRI BallTouches counter: exact, but only replicated
	// by recent game versions.
	Touches []touch
	// Index in Snaps of the first in-play frame after each kickoff countdown.
	KickoffStarts []int
	Playlist      int
}

type body struct {
	Pos, Vel vec
}

type boostComp struct {
	car    int
	amount float64
	active bool
}

// decode runs rrrocket on the replay to get its network frames.
func decode(path string) (*netReplay, error) {
	out, err := exec.Command("rrrocket", "-n", path).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("rrrocket: %v: %s", err, ee.Stderr)
		}
		return nil, fmt.Errorf("rrrocket: %w", err)
	}
	var nr netReplay
	if err := json.Unmarshal(out, &nr); err != nil {
		return nil, fmt.Errorf("rrrocket output: %w", err)
	}
	return &nr, nil
}

func buildTimeline(nr *netReplay) *timeline {
	tl := &timeline{}
	actorObj := map[int]string{}
	priName := map[int]string{}
	priTeam := map[int]int{} // PRI -> team actor
	priTouches := map[int]int{}
	teamNum := map[int]int{} // team actor -> 0/1
	carPRI := map[int]int{}
	cars := map[int]*body{}
	comps := map[int]*boostComp{}
	ballID := -1
	var ball *body
	state := ""
	kickoffPending := false
	clock, overtime := 300, false
	seenDemo := map[[2]int]bool{}

	player := func(car int) (string, int, bool) {
		pri, ok := carPRI[car]
		if !ok || priName[pri] == "" {
			return "", 0, false
		}
		team, ok := teamNum[priTeam[pri]]
		return priName[pri], team, ok
	}

	for fi, f := range nr.NetworkFrames.Frames {
		for _, id := range f.DeletedActors {
			delete(actorObj, id) // a frame can delete and re-create the same id
			delete(cars, id)
			delete(comps, id)
			for k := range seenDemo {
				if k[1] == id { // actor ids get reused
					delete(seenDemo, k)
				}
			}
			if id == ballID {
				ballID, ball = -1, nil
			}
		}
		for _, a := range f.NewActors {
			obj := nr.Objects[a.ObjectID]
			prev, existed := actorObj[a.ActorID]
			actorObj[a.ActorID] = obj
			if existed && prev == obj {
				continue // keyframe re-sends live actors
			}
			switch {
			case strings.HasPrefix(obj, "Archetypes.Car."):
				cars[a.ActorID] = &body{}
			case strings.HasPrefix(obj, "Archetypes.Ball."):
				ballID, ball = a.ActorID, &body{}
			case obj == "Archetypes.CarComponents.CarComponent_Boost":
				comps[a.ActorID] = &boostComp{car: -1, amount: boostMax / 3}
			case strings.HasPrefix(obj, "Archetypes.Teams.Team"):
				teamNum[a.ActorID] = int(obj[len(obj)-1] - '0')
			}
		}

		for _, c := range comps {
			if c.active {
				c.amount = max(0, c.amount-boostDrainRate*f.Delta)
			}
		}

		for _, u := range f.UpdatedActors {
			at := u.Attribute
			name := nr.Objects[u.ObjectID]
			if i := strings.LastIndex(name, ":"); i >= 0 {
				name = name[i+1:]
			}
			switch {
			case name == "ReplicatedRBState" && at.RigidBody != nil:
				b := cars[u.ActorID]
				if u.ActorID == ballID {
					b = ball
				}
				if b != nil {
					b.Pos = at.RigidBody.Location
					b.Vel = vec{} // no velocity: the body is sleeping
					if v := at.RigidBody.LinearVelocity; v != nil {
						b.Vel = *v
					}
				}
			case name == "PlayerName" && at.String != nil:
				priName[u.ActorID] = alias(nr.aliases, *at.String)
			case name == "Team" && at.ActiveActor != nil:
				priTeam[u.ActorID] = at.ActiveActor.Actor
			case name == "PlayerReplicationInfo" && at.ActiveActor != nil && at.ActiveActor.Active:
				// a demolished car loses its PRI just before the demolish
				// attribute, so the last owner is kept
				carPRI[u.ActorID] = at.ActiveActor.Actor
			case name == "Vehicle" && at.ActiveActor != nil:
				if c := comps[u.ActorID]; c != nil {
					c.car = at.ActiveActor.Actor
				}
			case name == "ReplicatedActive" && at.Byte != nil:
				if c := comps[u.ActorID]; c != nil {
					c.active = *at.Byte%2 == 1
				}
			case name == "ReplicatedBoost" && at.ReplicatedBoost != nil,
				name == "ReplicatedBoostAmount" && at.Byte != nil:
				c := comps[u.ActorID]
				if c == nil {
					continue
				}
				v := 0
				if at.ReplicatedBoost != nil {
					v = at.ReplicatedBoost.BoostAmount
				} else {
					v = *at.Byte
				}
				// A jump above the simulated value is a pad pickup; a small
				// pad gives 12% (~31/255).
				if gain := float64(v) - c.amount; gain > 8 && state == "Active" {
					if p, _, ok := player(c.car); ok {
						tl.Pickups = append(tl.Pickups, pickup{Time: f.Time, Player: p, Big: gain > 45, Before: c.amount / boostMax * 100})
					}
				}
				c.amount = float64(v)
			case name == "ReplicatedStateName" && at.Int != nil:
				if *at.Int >= 0 && *at.Int < len(nr.Names) {
					state = nr.Names[*at.Int]
					if state == "Countdown" {
						kickoffPending = true
					}
				}
			case name == "ReplicatedGamePlaylist" && at.Int != nil:
				tl.Playlist = *at.Int
			case name == "SecondsRemaining" && at.Int != nil:
				clock = *at.Int
			case name == "bOverTime" && at.Boolean != nil:
				overtime = *at.Boolean
			case name == "BallTouches" && at.Int != nil:
				if *at.Int > priTouches[u.ActorID] && priName[u.ActorID] != "" {
					tl.Touches = append(tl.Touches, touch{Frame: fi, Time: f.Time, Player: priName[u.ActorID]})
				}
				priTouches[u.ActorID] = *at.Int
			}
			if d := firstNonNil(at.DemolishExtended, at.DemolishFx.demolish(), at.Demolish.demolish()); d != nil && d.Attacker.Active && d.Victim.Active {
				key := [2]int{d.Attacker.Actor, d.Victim.Actor}
				att, _, ok1 := player(d.Attacker.Actor)
				vic, _, ok2 := player(d.Victim.Actor)
				if !seenDemo[key] && ok1 && ok2 {
					seenDemo[key] = true
					tl.Demos = append(tl.Demos, demo{Time: f.Time, Clock: clock, Overtime: overtime, Attacker: att, Victim: vic})
				}
			}
		}

		if state != "Active" || ball == nil {
			continue
		}
		s := snapshot{Frame: fi, Time: f.Time, Delta: f.Delta, Clock: clock, Overtime: overtime, Ball: ball.Pos, BallVel: ball.Vel}
		boostOf := map[int]float64{}
		for _, c := range comps {
			boostOf[c.car] = c.amount / boostMax * 100
		}
		for id, b := range cars {
			if p, team, ok := player(id); ok {
				s.Cars = append(s.Cars, carSnap{Player: p, Team: team, Pos: b.Pos, Vel: b.Vel, Boost: boostOf[id]})
			}
		}
		if kickoffPending {
			kickoffPending = false
			tl.KickoffStarts = append(tl.KickoffStarts, len(tl.Snaps))
		}
		tl.Snaps = append(tl.Snaps, s)
	}
	return tl
}

func firstNonNil(ds ...*demolish) *demolish {
	for _, d := range ds {
		if d != nil {
			return d
		}
	}
	return nil
}
