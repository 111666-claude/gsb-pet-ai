# pet-ai

宠物/怪物行为树：目标过滤、技能冷却与卡死处理，只用 Go 标准库。

```
go test ./...
go run ./cmd/pet-ai --sample invuln
go run ./cmd/pet-ai --sample stuck
```

## 口径（README 为准）

- **目标过滤**：无敌单位不能作为攻击目标，超出 `AttackRange` 的目标也要跳过。
  `--sample invuln`（无敌的 ghost 在前、可打的 real 在后）按口径是 `action=attack real`，
  现在只看 HP，给 `action=attack ghost`。
- **卡死要换目标**：连续 `StuckFrames` 帧都无法接近目标（距离一直超出攻击范围）就要换目标或进入 idle，
  并把 `Switches` 加一。`--sample stuck`（5 帧都只看到 Distance=999 的目标）按口径是 `switches >= 1`，
  现在给 `switches=0`。
- **技能冷却按时间**：冷却按毫秒累积（不是帧计数），帧率变化不影响 CD。
- 不变量：永不攻击无敌或死亡目标；`Switches` 单调不减；同一批帧重复回放结果相同。
- 规模：单帧 1 万个可选目标 0.5 秒内完成，内存 O(目标数)；不许每帧全表排序。

## 实现要点

- `Tick`/`TickDelta` 过滤 `HP <= 0`、无敌与超出 `AttackRange` 的攻击目标。
- 冷却按毫秒累积（`CooldownMS`），帧率变化不影响 CD。
- 连续 `StuckFrames` 帧够不到目标就换目标或进入 idle，`Switches` 加一。

## 输出

```
action=attack real
action=attack far switches=1
```

## 目录

```
brain.go            行为树
cmd/pet-ai          命令行入口
brain_test.go       go test 用例
```
