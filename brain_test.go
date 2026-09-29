package petai

import "testing"

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
