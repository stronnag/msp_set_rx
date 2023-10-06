package main

import (
	"encoding/binary"
	"fmt"
	"github.com/mattn/go-tty"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	ASTATE_Unknown = iota
	ASTATE_Ready
	ASTATE_Armed
	ASTATE_Disarmed
)

func get_status(v SChan) (status uint64, armflags uint32) {
	if v.cmd == msp2_INAV_STATUS {
		status = binary.LittleEndian.Uint64(v.data[13:21])
	} else {
		status = uint64(binary.LittleEndian.Uint32(v.data[6:10]))
	}

	if v.cmd == msp_STATUS_EX {
		armflags = uint32(binary.LittleEndian.Uint16(v.data[13:15]))
	} else {
		armflags = binary.LittleEndian.Uint32(v.data[9:13])
	}
	return status, armflags
}

func (m *MSPSerial) find_status_cmd() (stscmd uint16) {
	// MSP STatus inquiry, INAV version dependent
	if m.vcapi > 0x200 {
		if m.fcvers >= 0x010801 {
			stscmd = msp2_INAV_STATUS
		} else {
			stscmd = msp_STATUS_EX
		}
	} else {
		stscmd = msp_STATUS
	}
	return stscmd
}

func (m *MSPSerial) test_rx(setthr int, verbose bool) {
	phase := PHASE_Quiescent
	stscmd := m.find_status_cmd()
	fs := false
	xboxflags := uint64(0)
	xarmflags := uint32(0)
	dpending := false

	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	evchan := make(chan rune)
	go func() {
		for {
			r, err := tty.ReadRune()
			if err != nil {
				log.Panic(err)
			}
			evchan <- r
		}
	}()

	cc := make(chan os.Signal, 1)
	signal.Notify(cc, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Keypresses: 'A'/'a': toggle arming, 'Q'/'q': quit, 'F': quit to failsafe")
	if setthr > 999 && setthr < 2001 {
		fmt.Println("            '+'/'-' raise / lower throttle by 25µs")
	}
	log.Printf("Start TX loop")

	ticker := time.NewTicker(100 * time.Millisecond)

	for done := false; done == false; {
		select {
		case <-ticker.C:
			tdata := m.serialise_rx(phase, setthr, fs)
			m.Send_msp(msp_SET_RAW_RC, tdata)
			if verbose {
				txdata := deserialise_rx(tdata)
				fmt.Printf("Tx: %v\n", txdata)
			}
		case v := <-m.c0:
			if v.ok {
				switch v.cmd {
				case msp_SET_RAW_RC:
					if verbose {
						m.Send_msp(msp_RC, nil)
					} else {
						m.Send_msp(stscmd, nil)
					}
				case msp_RC:
					rxdata := deserialise_rx(v.data)
					fmt.Printf("Rx: %v\n", rxdata)
					m.Send_msp(stscmd, nil)

				case msp2_INAV_STATUS, msp_STATUS_EX, msp_STATUS:
					boxflags, armflags := get_status(v)
					// Unarmed, able to arm
					if boxflags != xboxflags || xarmflags != armflags {
						log.Printf("Box: %s (%x) Arm: %s\n", m.format_box(boxflags), boxflags, arm_status(armflags))
						fs = ((boxflags & m.fail_mask) == m.fail_mask)
						if (boxflags&m.arm_mask == 0) && armflags < 0x80 {
							phase = PHASE_Quiescent
						}

						if (boxflags & m.arm_mask) == 0 { // Disarmed
							phase = PHASE_Quiescent
							done = dpending
						} else {
							phase = PHASE_LowThrottle // Armed
						}
						xboxflags = boxflags
						xarmflags = armflags
					}
				default:
				}
			} else {
				log.Printf("MSP %d (%x) failed\n", v.cmd, v.cmd)
				done = true
			}

		case ev := <-evchan:
			switch ev {
			case 'A', 'a':
				switch phase {
				case PHASE_Quiescent:
					phase = PHASE_Arming
				case PHASE_LowThrottle:
					phase = PHASE_Disarming
				default:
				}
			case 'F':
				done = true
			case 'Q', 'q':
				phase, done, dpending = safe_quit(phase)
			case '+', '-':
				if setthr > 999 && setthr < 2001 {
					if ev == '+' {
						setthr += 25
					} else {
						setthr -= 25
					}
					if setthr > 2000 {
						setthr = 2000
					} else if setthr < 1000 {
						setthr = 1000
					}
					fmt.Printf("Throttle: %d\n", setthr)
				}
			}
		case <-cc:
			log.Println("Interrupt")
			phase, done, dpending = safe_quit(phase)
		}
	}
}

func safe_quit(phase int) (int, bool, bool) {
	dpending := false
	done := false
	if phase == PHASE_LowThrottle || phase == PHASE_Disarming {
		dpending = true
		phase = PHASE_Disarming
	} else {
		done = true
	}
	return phase, done, dpending
}

func arm_status(status uint32) string {
	armfails := [...]string{
		"",           /*      1 */
		"",           /*      2 */
		"Armed",      /*      4 */
		"Ever armed", /*      8 */
		"",           /*     10 */ // HITL
		"",           /*     20 */ // SITL
		"",           /*     40 */
		"F/S",        /*     80 */
		"Level",      /*    100 */
		"Calibrate",  /*    200 */
		"Overload",   /*    400 */
		"NavUnsafe", "MagCal", "AccCal", "ArmSwitch", "H/WFail",
		"BoxF/S", "BoxKill", "RCLink", "Throttle", "CLI",
		"CMS", "OSD", "Roll/Pitch", "Autotrim", "OOM",
		"Settings", "PWM Out", "PreArm", "DSHOTBeep", "Land", "Other",
	}

	var sarry []string
	if status < 0x80 {
		if status&(1<<2) != 0 {
			sarry = append(sarry, armfails[2])
		}
		if len(sarry) == 0 {
			sarry = append(sarry, "Ready to arm")
		}
	} else {
		for i := 0; i < len(armfails); i++ {
			if ((status & (1 << i)) != 0) && armfails[i] != "" {
				sarry = append(sarry, armfails[i])
			}
		}
	}
	sarry = append(sarry, fmt.Sprintf("(0x%x)", status))
	return strings.Join(sarry, " ")
}
