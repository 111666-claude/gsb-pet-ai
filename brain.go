// Package petai 是宠物/怪物行为树：挑目标、放技能、卡住换目标。
package petai

// AttackRange 是攻击距离。
const AttackRange = 5.0

// StuckFrames 是判定卡住需要的帧数。
const StuckFrames = 3

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

	// TimeMS 是当前时间（毫秒），由调用方在每帧前推进；
	// 冷却按它累积，帧率变化不影响 CD。
	TimeMS int64
	// CooldownMS 是攻击冷却时长（毫秒），0 表示无冷却。
	CooldownMS int64

	stuck           int
	cooldownUntilMS int64
}

// Tick 跑一帧：过滤无敌/死亡/超距目标，按毫秒结算冷却，
// 连续 StuckFrames 帧无法接近目标就换目标或进入 idle。
// 单帧 O(len(actors)) 一次扫描，不做全表排序。
func (b *Brain) Tick(actors []Actor) string {
	b.Frame++

	var firstCandidate *Actor // 第一个存活且非无敌的目标
	var firstInRange *Actor   // 第一个还在攻击范围内的目标
	var fallback *Actor       // 第一个非当前目标的候选（卡死时换目标用）
	currentValid := false
	for i := range actors {
		actor := &actors[i]
		if actor.HP <= 0 || actor.Invulnerable {
			continue
		}
		if firstCandidate == nil {
			firstCandidate = actor
		}
		if actor.Name == b.Target {
			currentValid = true
		} else if fallback == nil {
			fallback = actor
		}
		if firstInRange == nil && actor.Distance <= AttackRange {
			firstInRange = actor
		}
	}

	if firstInRange != nil {
		b.stuck = 0
		if b.TimeMS < b.cooldownUntilMS {
			return "idle" // 技能冷却中
		}
		b.Target = firstInRange.Name
		b.cooldownUntilMS = b.TimeMS + b.CooldownMS
		return "attack " + b.Target
	}

	if !currentValid {
		b.stuck = 0
		if firstCandidate == nil {
			b.Target = ""
			return "idle"
		}
		b.Target = firstCandidate.Name
	}

	// 目标在攻击范围外，这一帧无法接近。
	b.stuck++
	if b.stuck >= StuckFrames {
		b.Switches++
		b.stuck = 0
		if fallback != nil {
			b.Target = fallback.Name
			return "attack " + b.Target
		}
		b.Target = ""
		return "idle"
	}
	return "attack " + b.Target
}
