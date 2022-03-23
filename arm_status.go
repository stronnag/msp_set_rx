package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	arm_fails := [...]string{"", "", "Armed", "", "", "", "",
		"Failsafe", "Not level", "Calibrating", "Overload",
		"Navigation unsafe", "Compass cal", "Acc cal", "Arm switch", "Hardware failure",
		"Box failsafe", "Box killswitch", "RC Link", "Throttle", "CLI",
		"CMS Menu", "OSD Menu", "Roll/Pitch", "Servo Autotrim", "Out of memory",
		"Settings", "PWM Output", "PreArm", "DSHOTBeeper", "Other"}

	if len(os.Args) == 2 {
		if v, err := strconv.ParseInt(os.Args[1], 16, 64); err == nil {
			for i := 0; i < 32; i++ {
				if (v & (1 << i)) != 0 {
					if arm_fails[i] != "" {
						fmt.Printf("%08x => %s\n", (1 << i), arm_fails[i])
					}
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "Failed to parse \"%s\" as a valid status hexadecimal value\n", os.Args[1])
		}
	} else {
		fmt.Fprintln(os.Stderr, "Require precisely one integer argument")
	}
}
