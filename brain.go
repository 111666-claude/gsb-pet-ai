// Package petai 是宠物/怪物行为树：挑目标、放技能、卡住换目标。
// 缺陷：目标选择不过滤无敌与超距、技能冷却用帧计数、卡死换目标没实现。
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
}

// Tick 跑一帧。缺陷：只看 HP，不看无敌与距离；也不会因为卡住而换目标。
func (b *Brain) Tick(actors []Actor) string {
	b.Frame++
	for _, actor := range actors {
		if actor.HP <= 0 {
			continue
		}
		b.Target = actor.Name
		return "attack " + actor.Name
	}
	return "idle"
}
