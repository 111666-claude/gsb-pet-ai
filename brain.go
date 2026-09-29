// Package petai 是宠物/怪物行为树：挑目标、放技能、卡住换目标。
// 口径见 README：过滤无敌/死亡/超距目标，冷却按毫秒累积，卡死换目标并计 Switches。
package petai

// AttackRange 是攻击距离。
const AttackRange = 5.0

// StuckFrames 是判定卡住需要的帧数。
const StuckFrames = 3

// CooldownMS 是技能冷却毫秒数。冷却按时间累积，帧率变化不影响 CD。
const CooldownMS = 1000

// DefaultFrameMS 是 Tick 缺省的帧间隔毫秒数。
const DefaultFrameMS = 50

// Actor 是一个可选目标。
type Actor struct {
	Name         string
	HP           int
	Invulnerable bool
	Distance     float64
}

// Brain 是行为树状态。
type Brain struct {
	Target   string
	Frame    int
	Switches int

	stuck      int
	cooldownMS int
}

// Tick 按缺省帧间隔跑一帧。
func (b *Brain) Tick(actors []Actor) string {
	return b.TickDelta(actors, DefaultFrameMS)
}

// TickDelta 跑一帧，deltaMS 是距上一帧经过的毫秒数。
// 单趟扫描选目标，O(目标数) 时间、O(1) 额外内存，不做全表排序。
func (b *Brain) TickDelta(actors []Actor, deltaMS int) string {
	b.Frame++
	if deltaMS < 0 {
		deltaMS = 0
	}
	b.cooldownMS -= deltaMS
	if b.cooldownMS < 0 {
		b.cooldownMS = 0
	}

	// 当前目标死亡/无敌/消失则丢弃。
	if b.Target != "" {
		cur := findActor(actors, b.Target)
		if cur == nil || cur.HP <= 0 || cur.Invulnerable {
			b.Target, b.stuck = "", 0
		}
	}

	// 没有锁定时选最近的存活非无敌目标（追击目标，攻击仍要求进入射程）。
	if b.Target == "" {
		b.Target = selectTarget(actors, "")
	}

	// 连续超距累计卡死，满 StuckFrames 帧就换目标或进入 idle。
	if b.Target != "" {
		if cur := findActor(actors, b.Target); cur != nil && cur.Distance > AttackRange {
			b.stuck++
			if b.stuck >= StuckFrames {
				b.Switches++
				b.Target = selectTarget(actors, b.Target)
				b.stuck = 0
			}
		} else {
			b.stuck = 0
		}
	}
	if b.Target == "" {
		return "idle"
	}

	cur := findActor(actors, b.Target)
	if cur == nil || cur.Distance > AttackRange || b.cooldownMS > 0 {
		return "idle"
	}
	b.cooldownMS = CooldownMS
	return "attack " + b.Target
}

// findActor 按名字找目标，找不到返回 nil。
func findActor(actors []Actor, name string) *Actor {
	if name == "" {
		return nil
	}
	for i := range actors {
		if actors[i].Name == name {
			return &actors[i]
		}
	}
	return nil
}

// selectTarget 单趟选出最近的存活非无敌目标，并列时取列表靠前者。
func selectTarget(actors []Actor, exclude string) string {
	best := ""
	bestDist := 0.0
	for i := range actors {
		a := &actors[i]
		if a.HP <= 0 || a.Invulnerable || a.Name == exclude {
			continue
		}
		if best == "" || a.Distance < bestDist {
			best, bestDist = a.Name, a.Distance
		}
	}
	return best
}
