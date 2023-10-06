package main

import (
	"encoding/binary"
	"fmt"
	"go.bug.st/serial"
	"log"
	"math/rand"
	"net"
	"os"
	"sort"
	"strings"
)

const (
	PERM_ARM     = 0
	PERM_MANUAL  = 12
	PERM_HORIZON = 2
	PERM_ANGLE   = 1
	PERM_LAUNCH  = 36
	PERM_RTH     = 10
	PERM_WP      = 28
	PERM_CRUISE  = 45
	PERM_ALTHOLD = 3
	PERM_POSHOLD = 11
	PERM_FS      = 27
)

const (
	PHASE_Unknown = iota
	PHASE_Quiescent
	PHASE_Arming
	PHASE_LowThrottle
	PHASE_Disarming
)

const (
	msp_API_VERSION = 1
	msp_FC_VARIANT  = 2
	msp_FC_VERSION  = 3
	msp_BOARD_INFO  = 4
	msp_BUILD_INFO  = 5

	msp_NAME        = 10
	msp_MODE_RANGES = 34
	msp_STATUS      = 101
	msp_SET_RAW_RC  = 200
	msp_RC          = 105
	msp_STATUS_EX   = 150
	msp_RX_MAP      = 64
	msp_BOXNAMES    = 116

	msp_COMMON_SETTING = 0x1003
	msp2_INAV_STATUS   = 0x2000
	rx_START           = 1400
	rx_RAND            = 200
)
const (
	state_INIT = iota
	state_M
	state_DIRN
	state_LEN
	state_CMD
	state_DATA
	state_CRC

	state_X_HEADER2
	state_X_FLAGS
	state_X_ID1
	state_X_ID2
	state_X_LEN1
	state_X_LEN2
	state_X_DATA
	state_X_CHECKSUM
)

const SETTING_STR string = "nav_extra_arming_safety"
const MAX_MODE_ACTIVATION_CONDITION_COUNT int = 40

type SChan struct {
	len  uint16
	cmd  uint16
	ok   bool
	data []byte
}

type SerDev interface {
	Read(buf []byte) (int, error)
	Write(buf []byte) (int, error)
	Close() error
}

type ModeRange struct {
	boxid   byte
	chanidx byte
	start   byte
	end     byte
}

type MSPSerial struct {
	klass     int
	sd        SerDev
	usev2     bool
	bypass    bool
	vcapi     uint16
	fcvers    uint32
	a         int8
	e         int8
	r         int8
	t         int8
	c0        chan SChan
	swchan    int8
	swvalue   uint16
	mranges   []ModeRange
	arm_mask  uint64
	fail_mask uint64
	boxparts  []string
}

var nchan = int(18)

func crc8_dvb_s2(crc byte, a byte) byte {
	crc ^= a
	for i := 0; i < 8; i++ {
		if (crc & 0x80) != 0 {
			crc = (crc << 1) ^ 0xd5
		} else {
			crc = crc << 1
		}
	}
	return crc
}

func encode_msp2(cmd uint16, payload []byte) []byte {
	var paylen int16
	if len(payload) > 0 {
		paylen = int16(len(payload))
	}
	buf := make([]byte, 9+paylen)
	buf[0] = '$'
	buf[1] = 'X'
	buf[2] = '<'
	buf[3] = 0 // flags
	binary.LittleEndian.PutUint16(buf[4:6], cmd)
	binary.LittleEndian.PutUint16(buf[6:8], uint16(paylen))
	if paylen > 0 {
		copy(buf[8:], payload)
	}
	crc := byte(0)
	for _, b := range buf[3 : paylen+8] {
		crc = crc8_dvb_s2(crc, b)
	}
	buf[8+paylen] = crc
	return buf
}

func encode_msp(cmd uint16, payload []byte) []byte {
	var paylen byte
	if len(payload) > 0 {
		paylen = byte(len(payload))
	}
	buf := make([]byte, 6+paylen)
	buf[0] = '$'
	buf[1] = 'M'
	buf[2] = '<'
	buf[3] = paylen
	buf[4] = byte(cmd)
	if paylen > 0 {
		copy(buf[5:], payload)
	}
	crc := byte(0)
	for _, b := range buf[3:] {
		crc ^= b
	}
	buf[5+paylen] = crc
	return buf
}

func (m *MSPSerial) Read_msp(c0 chan SChan) {
	inp := make([]byte, 1024)
	var sc SChan
	var count = uint16(0)
	var crc = byte(0)

	n := state_INIT

	for {
		nb, err := m.sd.Read(inp)
		if err == nil && nb > 0 {
			for i := 0; i < nb; i++ {
				switch n {
				case state_INIT:
					if inp[i] == '$' {
						n = state_M
						sc.ok = false
						sc.len = 0
						sc.cmd = 0
					}
				case state_M:
					if inp[i] == 'M' {
						n = state_DIRN
					} else if inp[i] == 'X' {
						n = state_X_HEADER2
					} else {
						n = state_INIT
					}
				case state_DIRN:
					if inp[i] == '!' {
						n = state_LEN
					} else if inp[i] == '>' {
						n = state_LEN
						sc.ok = true
					} else {
						n = state_INIT
					}

				case state_X_HEADER2:
					if inp[i] == '!' {
						n = state_X_FLAGS
					} else if inp[i] == '>' {
						n = state_X_FLAGS
						sc.ok = true
					} else {
						n = state_INIT
					}

				case state_X_FLAGS:
					crc = crc8_dvb_s2(0, inp[i])
					n = state_X_ID1

				case state_X_ID1:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.cmd = uint16(inp[i])
					n = state_X_ID2

				case state_X_ID2:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.cmd |= (uint16(inp[i]) << 8)
					n = state_X_LEN1

				case state_X_LEN1:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.len = uint16(inp[i])
					n = state_X_LEN2

				case state_X_LEN2:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.len |= (uint16(inp[i]) << 8)
					if sc.len > 0 {
						n = state_X_DATA
						count = 0
						sc.data = make([]byte, sc.len)
					} else {
						n = state_X_CHECKSUM
					}
				case state_X_DATA:
					crc = crc8_dvb_s2(crc, inp[i])
					sc.data[count] = inp[i]
					count++
					if count == sc.len {
						n = state_X_CHECKSUM
					}

				case state_X_CHECKSUM:
					ccrc := inp[i]
					if crc != ccrc {
						fmt.Fprintf(os.Stderr, "CRC error on %d\n", sc.cmd)
					} else {
						c0 <- sc
					}
					n = state_INIT

				case state_LEN:
					sc.len = uint16(inp[i])
					crc = inp[i]
					n = state_CMD
				case state_CMD:
					sc.cmd = uint16(inp[i])
					crc ^= inp[i]
					if sc.len == 0 {
						n = state_CRC
					} else {
						sc.data = make([]byte, sc.len)
						n = state_DATA
						count = 0
					}
				case state_DATA:
					sc.data[count] = inp[i]
					crc ^= inp[i]
					count++
					if count == sc.len {
						n = state_CRC
					}
				case state_CRC:
					ccrc := inp[i]
					if crc != ccrc {
						fmt.Fprintf(os.Stderr, "CRC error on %d\n", sc.cmd)
					} else {
						//						fmt.Fprintf(os.Stderr, "Cmd %v Len %v\n", sc.cmd, sc.len)
						c0 <- sc
					}
					n = state_INIT
				}
			}
		} else {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Read %v\n", err)
			} else {
				fmt.Fprintln(os.Stderr, "serial EOF")
			}
			m.sd.Close()
			os.Exit(2)
		}
	}
}

func NewMSPSerial(dd DevDescription) *MSPSerial {
	m := MSPSerial{swchan: -1, klass: dd.klass}
	switch dd.klass {
	case DevClass_SERIAL:
		p, err := serial.Open(dd.name, &serial.Mode{BaudRate: dd.param})
		if err != nil {
			log.Fatal(err)
		}
		m.sd = p
		return &m
	case DevClass_BT:
		bt := NewBT(dd.name)
		m.sd = bt
		return &m
	case DevClass_TCP:
		var conn net.Conn
		remote := fmt.Sprintf("%s:%d", dd.name, dd.param)
		addr, err := net.ResolveTCPAddr("tcp", remote)
		if err == nil {
			conn, err = net.DialTCP("tcp", nil, addr)
		}
		if err != nil {
			log.Fatal(err)
		}
		m.sd = conn
		return &m
	case DevClass_UDP:
		var laddr, raddr *net.UDPAddr
		var conn net.Conn
		var err error
		if dd.param1 != 0 {
			raddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name1, dd.param1))
			laddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name, dd.param))
		} else {
			if dd.name == "" {
				laddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name, dd.param))
			} else {
				raddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", dd.name, dd.param))
			}
		}
		if err == nil {
			conn, err = net.DialUDP("udp", laddr, raddr)
		}
		if err != nil {
			log.Fatal(err)
		}
		m.sd = conn
		return &m
	default:
		fmt.Fprintln(os.Stderr, "Unsupported device")
		os.Exit(1)
	}
	return nil
}

func (m *MSPSerial) Send_msp(cmd uint16, payload []byte) {
	var buf []byte
	if m.usev2 || cmd > 255 {
		buf = encode_msp2(cmd, payload)
	} else {
		buf = encode_msp(cmd, payload)
	}
	m.sd.Write(buf)
}

func MSPInit(dd DevDescription) *MSPSerial {
	var fw, api, vers, board, gitrev string
	var v6 bool

	m := NewMSPSerial(dd)
	m.c0 = make(chan SChan)
	go m.Read_msp(m.c0)

	m.Send_msp(msp_API_VERSION, nil)
	for done := false; !done; {
		select {
		case v := <-m.c0:
			switch v.cmd {
			case msp_API_VERSION:
				if v.len > 2 {
					api = fmt.Sprintf("%d.%d", v.data[1], v.data[2])
					m.vcapi = uint16(v.data[1])<<8 | uint16(v.data[2])
					m.usev2 = (v.data[1] == 2)
					m.Send_msp(msp_FC_VARIANT, nil)
				}
			case msp_FC_VARIANT:
				fw = string(v.data[0:4])
				m.Send_msp(msp_FC_VERSION, nil)
			case msp_FC_VERSION:
				vers = fmt.Sprintf("%d.%d.%d", v.data[0], v.data[1], v.data[2])
				m.fcvers = uint32(v.data[0])<<16 | uint32(v.data[1])<<8 | uint32(v.data[2])
				m.Send_msp(msp_BUILD_INFO, nil)
				v6 = (v.data[0] >= 6)
				if v.data[0] == 1 {
					nchan = 16
				}
			case msp_BUILD_INFO:
				gitrev = string(v.data[19:])
				m.Send_msp(msp_BOARD_INFO, nil)
			case msp_BOARD_INFO:
				if v.len > 8 {
					board = string(v.data[9:])
				} else {
					board = string(v.data[0:4])
				}
				fmt.Fprintf(os.Stderr, "%s v%s %s (%s) API %s\n", fw, vers, board, gitrev, api)
				if m.usev2 {
					lstr := len(SETTING_STR)
					buf := make([]byte, lstr+1)
					copy(buf, SETTING_STR)
					buf[lstr] = 0
					m.Send_msp(msp_COMMON_SETTING, buf)
				} else {
					m.Send_msp(msp_RX_MAP, nil)
				}

			case msp_COMMON_SETTING:
				if v.len > 0 {
					bystr := v.data[0]
					if v6 {
						bystr++
					}
					if bystr != 0 {
						m.bypass = true
					}
					fmt.Printf("%s: %d (bypass %v)\n", SETTING_STR, bystr, m.bypass)
				}
				m.Send_msp(msp_RX_MAP, nil)

			case msp_RX_MAP:
				if v.len == 4 {
					m.a = int8(v.data[0]) * 2
					m.e = int8(v.data[1]) * 2
					m.r = int8(v.data[2]) * 2
					m.t = int8(v.data[3]) * 2
					var cmap [4]byte
					cmap[v.data[0]] = 'A'
					cmap[v.data[1]] = 'E'
					cmap[v.data[2]] = 'R'
					cmap[v.data[3]] = 'T'
					fmt.Fprintf(os.Stderr, "map: %s\n", cmap)
				}
				m.Send_msp(msp_NAME, nil)
			case msp_NAME:
				if v.len > 0 {
					fmt.Fprintf(os.Stderr, "name: \"%s\"\n", v.data[:v.len])
				}
				m.Send_msp(msp_BOXNAMES, nil)
			case msp_BOXNAMES:
				if v.len > 0 {
					fmt.Fprintf(os.Stderr, "box: %s\n", v.data[:v.len])
					m.setup_box_masks(string(v.data))
				} else {
					fmt.Fprintln(os.Stderr, "No Boxen")
				}
				m.Send_msp(msp_MODE_RANGES, nil)

			case msp_MODE_RANGES:
				if v.len > 0 {
					m.deserialise_modes(v.data)
				}
				done = true
			default:
				fmt.Fprintf(os.Stderr, "Unsolicited %d, length %d\n", v.cmd, v.len)
			}
		}
	}
	return m
}

func (m *MSPSerial) setup_box_masks(boxen string) {
	m.boxparts = strings.Split(boxen, ";")
	for i, b := range m.boxparts {
		switch b {
		case "ARM":
			m.arm_mask = 1 << i
		case "FAILSAFE":
			m.fail_mask = 1 << i
		default:
		}
	}
}

func (m *MSPSerial) format_box(bval uint64) string {
	var sb strings.Builder
	for i, b := range m.boxparts {
		j := uint64(1 << i)
		if (bval & j) != 0 {
			sb.WriteString(b)
			sb.WriteByte(',')
		}
	}
	if sb.Len() == 0 {
		return ""
	} else {
		return sb.String()[0 : sb.Len()-1]
	}
}

/*
	 for reference
			   type ModeRange struct {
			    boxid   byte 0
			    chanidx byte 1
			    start   byte 2
			    end     byte 3
		     }
*/
func (m *MSPSerial) deserialise_modes(buf []byte) {
	i := 0
	for j := 0; j < MAX_MODE_ACTIVATION_CONDITION_COUNT; j++ {
		if i >= len(buf) {
			break
		}
		if buf[i+3] != 0 {
			invalid := (buf[0] == PERM_ARM && (buf[i+3]-buf[i+2]) > 40)
			if !invalid {
				m.mranges = append(m.mranges, ModeRange{buf[i], buf[i+1], buf[i+2], buf[i+3]})
			}
		}
		i += 4
	}
	sort.Slice(m.mranges, func(i, j int) bool {
		if m.mranges[i].chanidx != m.mranges[j].chanidx {
			return m.mranges[i].chanidx < m.mranges[j].chanidx
		}
		return m.mranges[i].start < m.mranges[j].start
	})

	for _, r := range m.mranges {
		dump_mode(r)
		if r.boxid == PERM_ARM {
			m.swchan = 4 + int8(r.chanidx)
			m.swvalue = uint16(r.end+r.start)*25/2 + 900
		}
	}
}

func (m *MSPSerial) serialise_rx(phase int, setthr int, fs bool) []byte {
	buf := make([]byte, nchan*2)
	aoff := int(0)

	if m.swchan != -1 {
		aoff = int(m.swchan) * 2
	}

	var ae = m.a + 2
	var ee = m.e + 2
	var re = m.r + 2
	var te = m.t + 2

	for i := 4; i < nchan; i++ {
		binary.LittleEndian.PutUint16(buf[i*2:2+i*2], uint16(1000))
	}

	if aoff != 0 {
		binary.LittleEndian.PutUint16(buf[aoff:aoff+2], uint16(1001)) // a little clue as to the arm channel
	}

	baseval := uint16(1500)
	if fs {
		n := uint16(rand.Intn(rx_RAND))
		baseval += (n - rx_RAND/2)
	}

	switch phase {
	case PHASE_Unknown:
		n := rand.Intn(rx_RAND)
		binary.LittleEndian.PutUint16(buf[m.a:ae], uint16(rx_START+n))
		n = rand.Intn(rx_RAND)
		binary.LittleEndian.PutUint16(buf[m.e:ee], uint16(rx_START+n))
		n = rand.Intn(rx_RAND)
		binary.LittleEndian.PutUint16(buf[m.r:re], uint16(rx_START+n))
		n = rand.Intn(rx_RAND)
		binary.LittleEndian.PutUint16(buf[m.t:te], uint16(990))
	case PHASE_Quiescent:
		binary.LittleEndian.PutUint16(buf[m.a:ae], baseval)
		binary.LittleEndian.PutUint16(buf[m.e:ee], baseval)
		if m.bypass {
			binary.LittleEndian.PutUint16(buf[m.r:re], uint16(2000))
		} else {
			binary.LittleEndian.PutUint16(buf[m.r:re], uint16(1500))
		}
		binary.LittleEndian.PutUint16(buf[m.t:te], uint16(1000))

	case PHASE_Arming:
		binary.LittleEndian.PutUint16(buf[m.a:ae], baseval)
		binary.LittleEndian.PutUint16(buf[m.e:ee], baseval)
		if m.bypass {
			binary.LittleEndian.PutUint16(buf[m.r:re], uint16(1999))
		} else {
			binary.LittleEndian.PutUint16(buf[m.r:re], uint16(1501))
		}
		if aoff != 0 {
			binary.LittleEndian.PutUint16(buf[aoff:aoff+2], uint16(m.swvalue))
		}
		binary.LittleEndian.PutUint16(buf[m.t:te], uint16(1000))
	case PHASE_LowThrottle:
		binary.LittleEndian.PutUint16(buf[m.a:ae], baseval)
		binary.LittleEndian.PutUint16(buf[m.e:ee], baseval)
		binary.LittleEndian.PutUint16(buf[m.r:re], uint16(1500))
		thr := uint16(0)
		if setthr < 1000 {
			thr = uint16(1100 + rand.Intn(rx_RAND))
		} else {
			thr = uint16(setthr)
		}
		binary.LittleEndian.PutUint16(buf[m.t:te], uint16(thr))
		if aoff != 0 {
			binary.LittleEndian.PutUint16(buf[aoff:aoff+2], uint16(m.swvalue))
		}
	case PHASE_Disarming:
		binary.LittleEndian.PutUint16(buf[m.a:ae], baseval)
		binary.LittleEndian.PutUint16(buf[m.e:ee], baseval)
		binary.LittleEndian.PutUint16(buf[m.r:re], uint16(1500))
		binary.LittleEndian.PutUint16(buf[aoff:aoff+2], uint16(999))
		binary.LittleEndian.PutUint16(buf[m.t:te], uint16(1000))
	}
	return buf
}

func deserialise_rx(b []byte) []int16 {
	bl := binary.Size(b) / 2
	if bl > nchan {
		bl = nchan
	}
	buf := make([]int16, bl)
	for j := 0; j < bl; j++ {
		n := j * 2
		buf[j] = int16(binary.LittleEndian.Uint16(b[n : n+2]))
	}
	return buf
}
