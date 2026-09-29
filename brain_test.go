package petai

import (
	"math/rand"
	"strconv"
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
	action := brain.Tick([]Actor{
		{Name: "ghost", HP: 10, Invulnerable: true, Distance: 1},
		{Name: "real", HP: 10, Distance: 1},
	})
	if action != "attack real" {
		t.Fatalf("无敌目标必须被跳过：%s", action)
	}
}

func TestSkipsOutOfRange(t *testing.T) {
	brain := &Brain{}
	action := brain.Tick([]Actor{{Name: "far", HP: 10, Distance: AttackRange + 1}})
	if action != "idle" {
		t.Fatalf("超出 AttackRange 不能攻击：%s", action)
	}
}

func TestNeverAttacksInvulnerableOrDead(t *testing.T) {
	brain := &Brain{}
	frames := [][]Actor{
		{{Name: "ghost", HP: 10, Invulnerable: true, Distance: 1}},
		{{Name: "corpse", HP: 0, Distance: 1}},
		{{Name: "ghost", HP: 10, Invulnerable: true, Distance: 1}, {Name: "corpse", HP: -1, Distance: 2}},
	}
	for i, actors := range frames {
		if action := brain.Tick(actors); action != "idle" {
			t.Fatalf("第 %d 帧攻击了无敌或死亡目标：%s", i, action)
		}
	}
}

func TestStuckSwitchesTarget(t *testing.T) {
	brain := &Brain{}
	// 只有够不到的 far：锁定后连续超距，第 StuckFrames 帧触发换目标。
	for i := 0; i < StuckFrames-1; i++ {
		brain.Tick([]Actor{{Name: "far", HP: 10, Distance: 999}})
	}
	action := brain.Tick([]Actor{
		{Name: "far", HP: 10, Distance: 999},
		{Name: "near", HP: 10, Distance: 1},
	})
	if brain.Switches != 1 {
		t.Fatalf("卡死 %d 帧应换目标一次，Switches=%d", StuckFrames, brain.Switches)
	}
	if action != "attack near" {
		t.Fatalf("卡死后应换打 near：%s", action)
	}
}

func TestStuckFallsBackToIdle(t *testing.T) {
	brain := &Brain{}
	var action string
	for i := 0; i < StuckFrames+1; i++ {
		action = brain.Tick([]Actor{{Name: "far", HP: 10, Distance: 999}})
	}
	if brain.Switches < 1 {
		t.Fatalf("一直够不到应换目标，Switches=%d", brain.Switches)
	}
	if action != "idle" {
		t.Fatalf("没有别的目标应 idle：%s", action)
	}
}

func TestCooldownIsTimeBased(t *testing.T) {
	run := func(deltaMS, frames int) []string {
		brain := &Brain{}
		actions := make([]string, 0, frames)
		for i := 0; i < frames; i++ {
			actions = append(actions, brain.TickDelta([]Actor{{Name: "a", HP: 10, Distance: 1}}, deltaMS))
		}
		return actions
	}
	// 同样的总时长、不同帧率，攻击节奏必须一致。
	coarse := run(200, 6) // 6 帧 x 200ms = 1200ms
	fine := run(50, 24)   // 24 帧 x 50ms = 1200ms
	countAttacks := func(actions []string) int {
		n := 0
		for _, a := range actions {
			if a != "idle" {
				n++
			}
		}
		return n
	}
	if countAttacks(coarse) != countAttacks(fine) {
		t.Fatalf("帧率变化不应影响冷却：coarse=%v fine=%v", coarse, fine)
	}
	if countAttacks(coarse) != 2 {
		t.Fatalf("1200ms 内应攻击 2 次（0ms 与 1000ms）：%v", coarse)
	}
}

func TestSwitchesMonotonicAndReplayable(t *testing.T) {
	frames := [][]Actor{
		{{Name: "far", HP: 10, Distance: 999}},
		{{Name: "far", HP: 10, Distance: 999}},
		{{Name: "far", HP: 10, Distance: 999}},
		{{Name: "near", HP: 10, Distance: 1}},
		{{Name: "near", HP: 0, Distance: 1}, {Name: "ghost", HP: 5, Invulnerable: true, Distance: 1}},
	}
	replay := func() ([]string, int) {
		brain := &Brain{}
		actions := make([]string, 0, len(frames))
		lastSwitches := 0
		for _, actors := range frames {
			actions = append(actions, brain.Tick(actors))
			if brain.Switches < lastSwitches {
				t.Fatalf("Switches 必须单调不减：%d -> %d", lastSwitches, brain.Switches)
			}
			lastSwitches = brain.Switches
		}
		return actions, lastSwitches
	}
	first, _ := replay()
	second, _ := replay()
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("同一批帧重复回放结果必须相同：第 %d 帧 %s != %s", i, first[i], second[i])
		}
	}
}

// reference 是「过滤 + 冷却 + 卡死换目标」的独立参考实现，用于逐帧对照。
type reference struct {
	target     string
	switches   int
	stuck      int
	cooldownMS int
}

func (r *reference) tick(actors []Actor, deltaMS int) string {
	if deltaMS < 0 {
		deltaMS = 0
	}
	r.cooldownMS -= deltaMS
	if r.cooldownMS < 0 {
		r.cooldownMS = 0
	}

	byName := make(map[string]Actor, len(actors))
	for _, a := range actors {
		byName[a.Name] = a
	}

	if r.target != "" {
		if cur, ok := byName[r.target]; !ok || cur.HP <= 0 || cur.Invulnerable {
			r.target, r.stuck = "", 0
		}
	}
	if r.target == "" {
		r.target = pickNearest(actors, "")
	}
	if r.target != "" {
		if cur := byName[r.target]; cur.Distance > AttackRange {
			r.stuck++
			if r.stuck >= StuckFrames {
				r.switches++
				r.target = pickNearest(actors, r.target)
				r.stuck = 0
			}
		} else {
			r.stuck = 0
		}
	}
	if r.target == "" {
		return "idle"
	}
	cur := byName[r.target]
	if cur.Distance > AttackRange || r.cooldownMS > 0 {
		return "idle"
	}
	r.cooldownMS = CooldownMS
	return "attack " + r.target
}

func pickNearest(actors []Actor, exclude string) string {
	best, bestDist := "", 0.0
	for _, a := range actors {
		if a.HP <= 0 || a.Invulnerable || a.Name == exclude {
			continue
		}
		if best == "" || a.Distance < bestDist {
			best, bestDist = a.Name, a.Distance
		}
	}
	return best
}

func TestMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20260930))
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	for trial := 0; trial < 200; trial++ {
		frames := 1 + rng.Intn(12)
		brain := &Brain{}
		ref := &reference{}
		lastSwitches := 0
		for f := 0; f < frames; f++ {
			n := rng.Intn(len(names) + 1)
			actors := make([]Actor, n)
			perm := rng.Perm(len(names))
			for i := 0; i < n; i++ {
				actors[i] = Actor{
					Name:         names[perm[i]],
					HP:           rng.Intn(12) - 1,
					Invulnerable: rng.Intn(4) == 0,
					Distance:     float64(rng.Intn(12)),
				}
			}
			deltaMS := rng.Intn(401)
			got := brain.TickDelta(actors, deltaMS)
			want := ref.tick(actors, deltaMS)
			if got != want {
				t.Fatalf("trial %d 第 %d 帧与参考实现不一致：got=%s want=%s actors=%+v delta=%d",
					trial, f, got, want, actors, deltaMS)
			}
			if brain.Switches < lastSwitches {
				t.Fatalf("trial %d Switches 必须单调不减", trial)
			}
			lastSwitches = brain.Switches
			if got != "idle" {
				attacked := got[len("attack "):]
				for _, a := range actors {
					if a.Name == attacked && (a.HP <= 0 || a.Invulnerable || a.Distance > AttackRange) {
						t.Fatalf("trial %d 攻击了非法目标 %+v", trial, a)
					}
				}
			}
		}
	}
}

func TestScalesToTenThousandTargets(t *testing.T) {
	actors := make([]Actor, 10000)
	for i := range actors {
		actors[i] = Actor{
			Name:         "mob" + strconv.Itoa(i),
			HP:           10,
			Invulnerable: i%3 == 0,
			Distance:     float64(i % 1000),
		}
	}
	brain := &Brain{}
	start := time.Now()
	action := brain.Tick(actors)
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("单帧 1 万目标应在 0.5 秒内完成，实际 %s", elapsed)
	}
	if action != "attack mob1000" {
		t.Fatalf("应攻击最近的合法目标：%s", action)
	}
}
