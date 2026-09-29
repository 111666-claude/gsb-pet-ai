package petai

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestAttacksFirstAlive(t *testing.T) {
	brain := &Brain{}
	if action := brain.Tick([]Actor{{Name: "a", HP: 10, Distance: 1}}); action != "attack a" {
		t.Fatalf("应该攻击第一个存活目标：%s", action)
	}
}

func TestIdleWithoutTargets(t *testing.T) {
	brain := &Brain{}
	if action := brain.Tick([]Actor{{Name: "dead", HP: 0}}); action != "idle" {
		t.Fatalf("没有目标应该 idle：%s", action)
	}
}

func TestFrameAdvances(t *testing.T) {
	brain := &Brain{}
	brain.Tick(nil)
	brain.Tick(nil)
	if brain.Frame != 2 {
		t.Fatalf("帧数应该推进：%d", brain.Frame)
	}
}

func TestConstants(t *testing.T) {
	if AttackRange != 5 || StuckFrames != 3 {
		t.Fatal("常量被改了")
	}
}

func TestSkipsInvulnerable(t *testing.T) {
	brain := &Brain{}
	actors := []Actor{
		{Name: "ghost", HP: 10, Invulnerable: true, Distance: 1},
		{Name: "real", HP: 10, Distance: 1},
	}
	if action := brain.Tick(actors); action != "attack real" {
		t.Fatalf("无敌目标要被跳过：%s", action)
	}
}

func TestSkipsOutOfRange(t *testing.T) {
	brain := &Brain{}
	actors := []Actor{
		{Name: "far", HP: 10, Distance: 999},
		{Name: "near", HP: 10, Distance: 1},
	}
	if action := brain.Tick(actors); action != "attack near" {
		t.Fatalf("超出 AttackRange 的目标要被跳过：%s", action)
	}
}

func TestStuckSwitchesToIdle(t *testing.T) {
	brain := &Brain{}
	far := []Actor{{Name: "far", HP: 10, Distance: 999}}
	action := ""
	for i := 0; i < 5; i++ {
		action = brain.Tick(far)
	}
	if brain.Switches < 1 {
		t.Fatalf("连续 %d 帧无法接近要换目标或 idle：switches=%d", StuckFrames, brain.Switches)
	}
	if action != "attack far" {
		t.Fatalf("换目标后没有别的候选，应重新锁定唯一目标：%s", action)
	}
}

func TestStuckSwitchesTarget(t *testing.T) {
	brain := &Brain{}
	actors := []Actor{
		{Name: "a", HP: 10, Distance: 999},
		{Name: "b", HP: 10, Distance: 999},
	}
	for i := 0; i < StuckFrames; i++ {
		brain.Tick(actors)
	}
	if brain.Switches != 1 || brain.Target != "b" {
		t.Fatalf("卡死应换到另一个目标：target=%s switches=%d", brain.Target, brain.Switches)
	}
}

func TestCooldownUsesMilliseconds(t *testing.T) {
	brain := &Brain{CooldownMS: 100}
	near := []Actor{{Name: "a", HP: 10, Distance: 1}}

	brain.TimeMS = 0
	if action := brain.Tick(near); action != "attack a" {
		t.Fatalf("首次攻击不受冷却限制：%s", action)
	}
	brain.TimeMS = 50
	if action := brain.Tick(near); action != "idle" {
		t.Fatalf("冷却未结束不该攻击：%s", action)
	}
	brain.TimeMS = 100
	if action := brain.Tick(near); action != "attack a" {
		t.Fatalf("冷却按毫秒累积，到时就要就绪：%s", action)
	}
}

func TestCooldownIgnoresFrameRate(t *testing.T) {
	// 同样的 100ms 冷却，无论多少帧走完这 100ms，到点都必须就绪。
	run := func(steps int) []string {
		brain := &Brain{CooldownMS: 100}
		near := []Actor{{Name: "a", HP: 10, Distance: 1}}
		actions := []string{brain.Tick(near)} // t=0 首次攻击
		for i := 1; i <= steps; i++ {
			brain.TimeMS = int64(100 * i / steps)
			actions = append(actions, brain.Tick(near))
		}
		return actions
	}
	for _, actions := range [][]string{run(1), run(2), run(8)} {
		if actions[len(actions)-1] != "attack a" {
			t.Fatalf("100ms 后冷却必须就绪，与帧率无关：%v", actions)
		}
	}
}

func TestInvariantsOnReplay(t *testing.T) {
	frames := [][]Actor{
		{{Name: "ghost", HP: 10, Invulnerable: true, Distance: 1}, {Name: "real", HP: 10, Distance: 1}},
		{{Name: "real", HP: 10, Distance: 999}},
		{{Name: "real", HP: 10, Distance: 999}},
		{{Name: "real", HP: 0, Distance: 1}, {Name: "other", HP: 5, Distance: 2}},
	}
	run := func() []string {
		brain := &Brain{}
		var actions []string
		lastSwitches := 0
		for _, actors := range frames {
			action := brain.Tick(actors)
			if brain.Switches < lastSwitches {
				t.Fatalf("Switches 必须单调不减：%d -> %d", lastSwitches, brain.Switches)
			}
			lastSwitches = brain.Switches
			for _, a := range actors {
				if (a.HP <= 0 || a.Invulnerable) && action == "attack "+a.Name {
					t.Fatalf("永不攻击无敌或死亡目标：%s", action)
				}
			}
			actions = append(actions, action)
		}
		return actions
	}
	first, second := run(), run()
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("同一批帧重复回放结果必须相同：%v vs %v", first, second)
		}
	}
}

func TestScalesToTenThousandTargets(t *testing.T) {
	actors := make([]Actor, 10000)
	for i := range actors {
		actors[i] = Actor{Name: fmt.Sprintf("mob-%d", i), HP: 10, Distance: 999}
	}
	actors[9999].Distance = 1
	brain := &Brain{}
	start := time.Now()
	if action := brain.Tick(actors); action != "attack mob-9999" {
		t.Fatalf("要在 1 万个目标里挑出可打的：%s", action)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("单帧 1 万目标要在 0.5 秒内完成：%s", elapsed)
	}
}

// referenceBrain 是「过滤 + 冷却 + 卡死换目标」的参考实现，
// 用最直接的写法（先建候选表再逐个判断），用来和 Brain 逐帧对照。
type referenceBrain struct {
	target          string
	switches        int
	stuck           int
	timeMS          int64
	cooldownMS      int64
	cooldownUntilMS int64
}

func (r *referenceBrain) tick(actors []Actor) string {
	var candidates []Actor
	for _, a := range actors {
		if a.HP > 0 && !a.Invulnerable {
			candidates = append(candidates, a)
		}
	}
	for _, a := range candidates {
		if a.Distance > AttackRange {
			continue
		}
		r.stuck = 0
		if r.timeMS < r.cooldownUntilMS {
			return "idle"
		}
		r.target = a.Name
		r.cooldownUntilMS = r.timeMS + r.cooldownMS
		return "attack " + r.target
	}
	currentValid := false
	fallback := ""
	hasFallback := false
	for _, a := range candidates {
		if a.Name == r.target {
			currentValid = true
		} else if !hasFallback {
			fallback, hasFallback = a.Name, true
		}
	}
	if !currentValid {
		r.stuck = 0
		if len(candidates) == 0 {
			r.target = ""
			return "idle"
		}
		r.target = candidates[0].Name
	}
	r.stuck++
	if r.stuck >= StuckFrames {
		r.switches++
		r.stuck = 0
		if hasFallback {
			r.target = fallback
			return "attack " + r.target
		}
		r.target = ""
		return "idle"
	}
	return "attack " + r.target
}

func TestMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	deltas := []int64{0, 8, 16, 33, 100}
	for trial := 0; trial < 200; trial++ {
		cooldown := []int64{0, 50, 100}[rng.Intn(3)]
		brain := &Brain{CooldownMS: cooldown}
		ref := &referenceBrain{cooldownMS: cooldown}
		lastSwitches := 0
		for frame := 0; frame < 1+rng.Intn(30); frame++ {
			count := rng.Intn(len(names) + 1)
			perm := rng.Perm(len(names))[:count]
			actors := make([]Actor, 0, count)
			for _, idx := range perm {
				distance := []float64{1, 3, 5, 7, 999}[rng.Intn(5)]
				actors = append(actors, Actor{
					Name:         names[idx],
					HP:           []int{0, 1, 10}[rng.Intn(3)],
					Invulnerable: rng.Intn(10) == 0,
					Distance:     distance,
				})
			}
			delta := deltas[rng.Intn(len(deltas))]
			brain.TimeMS += delta
			ref.timeMS += delta
			got, want := brain.Tick(actors), ref.tick(actors)
			if got != want {
				t.Fatalf("trial %d frame %d：动作不一致 got=%s want=%s", trial, frame, got, want)
			}
			if brain.Switches != ref.switches {
				t.Fatalf("trial %d frame %d：Switches 不一致 got=%d want=%d", trial, frame, brain.Switches, ref.switches)
			}
			if brain.Switches < lastSwitches {
				t.Fatalf("trial %d frame %d：Switches 必须单调不减", trial, frame)
			}
			lastSwitches = brain.Switches
			for _, a := range actors {
				if (a.HP <= 0 || a.Invulnerable) && strings.HasPrefix(got, "attack ") && got == "attack "+a.Name {
					t.Fatalf("trial %d frame %d：永不攻击无敌或死亡目标：%s", trial, frame, got)
				}
			}
		}
	}
}
