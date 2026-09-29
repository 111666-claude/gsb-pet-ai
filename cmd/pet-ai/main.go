// Command pet-ai 回放内置样例，打印最后一帧的动作与换目标次数。
package main

import (
	"flag"
	"fmt"
	"os"

	petai "example.com/petai"
)

// frameMS 是样例回放时每帧推进的毫秒数。
const frameMS = 16

func main() {
	sample := flag.String("sample", "", "要回放的样例：invuln 或 stuck")
	flag.Parse()

	var frames [][]petai.Actor
	switch *sample {
	case "invuln":
		// 无敌的 ghost 在前、可打的 real 在后。
		frames = [][]petai.Actor{{
			{Name: "ghost", HP: 10, Invulnerable: true, Distance: 1},
			{Name: "real", HP: 10, Distance: 1},
		}}
	case "stuck":
		// 5 帧都只看到 Distance=999 的目标。
		for i := 0; i < 5; i++ {
			frames = append(frames, []petai.Actor{
				{Name: "far", HP: 10, Distance: 999},
			})
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: pet-ai --sample invuln|stuck")
		os.Exit(2)
	}

	brain := &petai.Brain{}
	action := ""
	for _, actors := range frames {
		brain.TimeMS += frameMS
		action = brain.Tick(actors)
	}

	fmt.Printf("action=%s", action)
	if brain.Switches > 0 {
		fmt.Printf(" switches=%d", brain.Switches)
	}
	fmt.Println()
}
