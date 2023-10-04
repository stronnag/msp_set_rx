package main

import (
	"fmt"
)

var permnames = []struct {
	name   string
	permid byte
}{
	{name: "ARM", permid: 0},
	{name: "ANGLE", permid: 1},
	{name: "HORIZON", permid: 2},
	{name: "NAV ALTHOLD", permid: 3},
	{name: "HEADING HOLD", permid: 5},
	{name: "HEADFREE", permid: 6},
	{name: "HEADADJ", permid: 7},
	{name: "CAMSTAB", permid: 8},
	{name: "NAV RTH", permid: 10},
	{name: "NAV POSHOLD", permid: 11},
	{name: "MANUAL", permid: 12},
	{name: "BEEPER", permid: 13},
	{name: "LEDS OFF", permid: 15},
	{name: "LIGHTS", permid: 16},
	{name: "OSD OFF", permid: 19},
	{name: "TELEMETRY", permid: 20},
	{name: "AUTO TUNE", permid: 21},
	{name: "BLACKBOX", permid: 26},
	{name: "FAILSAFE", permid: 27},
	{name: "NAV WP", permid: 28},
	{name: "AIR MODE", permid: 29},
	{name: "HOME RESET", permid: 30},
	{name: "GCS NAV", permid: 31},
	{name: "FPV ANGLE MIX", permid: 32},
	{name: "SURFACE", permid: 33},
	{name: "FLAPERON", permid: 34},
	{name: "TURN ASSIST", permid: 35},
	{name: "NAV LAUNCH", permid: 36},
	{name: "SERVO AUTOTRIM", permid: 37},
	{name: "KILLSWITCH", permid: 38},
	{name: "CAMERA CONTROL 1", permid: 39},
	{name: "CAMERA CONTROL 2", permid: 40},
	{name: "CAMERA CONTROL 3", permid: 41},
	{name: "OSD ALT 1", permid: 42},
	{name: "OSD ALT 2", permid: 43},
	{name: "OSD ALT 3", permid: 44},
	{name: "NAV COURSE HOLD", permid: 45},
	{name: "MC BRAKING", permid: 46},
	{name: "USER1", permid: 47},
	{name: "USER2", permid: 48},
	{name: "USER3", permid: 57},
	{name: "USER4", permid: 58},
	{name: "LOITER CHANGE", permid: 49},
	{name: "MSP RC OVERRIDE", permid: 50},
	{name: "PREARM", permid: 51},
	{name: "TURTLE", permid: 52},
	{name: "NAV CRUISE", permid: 53},
	{name: "AUTO LEVEL TRIM", permid: 54},
	{name: "WP PLANNER", permid: 55},
	{name: "SOARING", permid: 56},
	{name: "MISSION CHANGE", permid: 59},
	{name: "BEEPER MUTE", permid: 60},
	{name: "MULTI FUNCTION", permid: 61},
	{name: "MIXER PROFILE 2", permid: 62},
	{name: "MIXER TRANSITION", permid: 63},
}

func mode_name(i uint8) string {
	for _, m := range permnames {
		if i == m.permid {
			return m.name
		}
	}
	return ""
}

func make_pwm(val uint8) uint16 {
	return 900 + uint16(val)*25
}

func dump_mode(r ModeRange) {
	mname := mode_name(r.boxid)
	minpwm := make_pwm(r.start)
	maxpwm := make_pwm(r.end)
	fmt.Printf("chan: %2d, start: %d, end: %d %s\n", r.chanidx+5, minpwm, maxpwm, mname)
}
