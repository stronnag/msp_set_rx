package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/tarm/serial"
	"log"
	"os"
	"net"
	"bufio"
	"time"
	"math/rand"
)

const (
	msp_API_VERSION = 1
	msp_FC_VARIANT  = 2
	msp_FC_VERSION  = 3
	msp_BOARD_INFO  = 4
	msp_BUILD_INFO  = 5

	msp_NAME        = 10
	msp_STATUS      = 101
	msp_SET_RAW_RC  = 200
  msp_RC          = 105
	msp_STATUS_EX   = 150

	rx_START = 1400
	rx_RAND  =  200

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

type MSPSerial struct {
	klass int
	p *serial.Port
	conn net.Conn
	reader *bufio.Reader
	usev2 bool
	vcapi uint16
}

func crc8_dvb_s2(crc byte, a byte) byte {
	crc ^= a
  for  i:= 0; i < 8; i++ {
    if (crc & 0x80) != 0 {
			crc = (crc << 1) ^ 0xd5
		} else {
      crc = crc << 1
    }
	}
  return crc
}

// In this example, cmd can be a byte, really its int16
func encode_msp2(cmd byte, payload []byte) []byte {
	var paylen int16
	if len(payload) > 0 {
		paylen = int16(len(payload))
	}
	buf := make([]byte, 9+paylen)
	buf[0] = '$'
	buf[1] = 'X'
	buf[2] = '<'
	buf[3] = 0 // flags
	binary.LittleEndian.PutUint16(buf[4:6], uint16(cmd))
	binary.LittleEndian.PutUint16(buf[6:8], uint16(paylen))
	if paylen > 0 {
		copy(buf[8:], payload)
	}
	crc := byte(0)
	for _, b := range buf[3:paylen+8] {
		crc = crc8_dvb_s2(crc, b)
	}
	buf[8+paylen] = crc
	return buf
}

func encode_msp(cmd byte, payload []byte) []byte {
	var paylen byte
	if len(payload) > 0 {
		paylen = byte(len(payload))
	}
	buf := make([]byte, 6+paylen)
	buf[0] = '$'
	buf[1] = 'M'
	buf[2] = '<'
	buf[3] = paylen
	buf[4] = cmd
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

func (m *MSPSerial) read(inp []byte) (int, error) {
	if m.klass == DevClass_SERIAL {
		return m.p.Read(inp)
	} else if m.klass == DevClass_TCP {
		return m.conn.Read(inp)
	} else {
		return m.reader.Read(inp)
	}
}

func (m *MSPSerial) write(payload []byte) (int, error) {
	if m.klass == DevClass_SERIAL {
		return m.p.Write(payload)
	} else {
		return m.conn.Write(payload)
	}
}

func (m *MSPSerial) Read_msp() (byte, []byte, error) {
	inp := make([]byte, 1)
	var count = byte(0)
	var len = byte(0)
	var crc = byte(0)
	var cmd = byte(0)
	var xlen = int16(0)
	var xcmd = int16(0)
	var xcount = int16(0)

	ok := byte(0)
	done := false
	var buf []byte
	var err error

	n := state_INIT

	for !done {
		_, err = m.read(inp)
		if err == nil {
			switch n {
			case state_INIT:
				if inp[0] == '$' {
					n = state_M
				}
			case state_M:
				if inp[0] == 'M' {
					n = state_DIRN
				} else if inp[0] == 'X'{
					n = state_X_HEADER2
				} else {
					n = state_INIT
				}
			case state_DIRN:
				if inp[0] == '!' {
					n = state_LEN
					ok = 1
				} else if inp[0] == '>' {
					n = state_LEN
				} else {
					n = state_INIT
				}

			case state_X_HEADER2:
				if inp[0] == '!' {
					n = state_X_FLAGS
					ok = 1
				} else if inp[0] == '>' {
					n = state_X_FLAGS
				} else {
					n = state_INIT
				}

			case state_X_FLAGS:
				crc = crc8_dvb_s2(0, inp[0])
				n = state_X_ID1

			case state_X_ID1:
				crc = crc8_dvb_s2(crc, inp[0])
				xcmd = int16(inp[0]);
				cmd = inp[0]
				n = state_X_ID2

			case state_X_ID2:
				crc = crc8_dvb_s2(crc, inp[0])
				xcmd |= (int16(inp[0])<<8);
				n = state_X_LEN1

			case state_X_LEN1:
				crc = crc8_dvb_s2(crc, inp[0])
				xlen = int16(inp[0])
				n = state_X_LEN2

			case state_X_LEN2:
				crc = crc8_dvb_s2(crc, inp[0])
				xlen |= (int16(inp[0])<<8);
				buf = make([]byte, xlen)
				if xlen > 0 {
					n = state_X_DATA
				} else {
					n = state_X_CHECKSUM
				}
			case state_X_DATA:
				crc = crc8_dvb_s2(crc, inp[0])
				buf[xcount] = inp[0]
				xcount++
				if xcount == xlen {
					n = state_X_CHECKSUM
				}

			case state_X_CHECKSUM:
				ccrc := inp[0]
				if crc != ccrc {
					ok = 2
				}
				done = true

			case state_LEN:
				len = inp[0]
				buf = make([]byte, len)
				crc = len
				n = state_CMD
			case state_CMD:
				cmd = inp[0]
				crc ^= cmd
				if len == 0 {
					n = state_CRC
				} else {
					n = state_DATA
				}
			case state_DATA:
				buf[count] = inp[0]
				crc ^= inp[0]
				count++
				if count == len {
					n = state_CRC
				}
			case state_CRC:
				ccrc := inp[0]
				if crc != ccrc {
					ok = 2
				}
				done = true
			}
		} else {
			done = true
		}
	}
	if ok != 0  {
		switch ok {
		case 1:
			err = errors.New("MSP unrecognised")
		case 2:
			err = errors.New("MSP CRC")
		}
		return cmd, nil, err
	} else {
		return cmd, buf, nil
	}
}

func (m *MSPSerial) Read_cmd(cmd byte) (byte, []byte, error) {
	var buf []byte
	var err error
	var c = byte(0)

	for ; c != cmd ; {
		c,buf,err = m.Read_msp()
		if c != cmd || err != nil {
			fmt.Printf("Received cmd %v (wanted %v) err=%v\n", c, cmd, err)
		}
	}
	return c,buf,err
}

func NewMSPSerial(dd DevDescription) *MSPSerial {
	c := &serial.Config{Name: dd.name, Baud: dd.param}
	p, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
	return &MSPSerial{klass: dd.klass, p: p}
}

func NewMSPTCP(dd DevDescription) *MSPSerial {
	var conn net.Conn
	remote := fmt.Sprintf("%s:%d", dd.name, dd.param)
	addr, err := net.ResolveTCPAddr("tcp", remote)
	if err == nil {
    conn, err = net.DialTCP("tcp", nil, addr)
	}

	if err != nil {
		log.Fatal(err)
	}
	return &MSPSerial{klass: dd.klass, conn: conn}
}

func NewMSPUDP(dd DevDescription) *MSPSerial {
	var laddr, raddr *net.UDPAddr
	var reader  *bufio.Reader
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
		conn,err = net.DialUDP("udp", laddr, raddr)
		if err == nil {
		reader = bufio.NewReader(conn)
		}
	}
	if err != nil {
		log.Fatal(err)
	}
	return &MSPSerial{klass: dd.klass, conn: conn, reader : reader}
}

func (m *MSPSerial) Send_msp(cmd byte, payload []byte) {
	var buf []byte
	if m.usev2 {
		buf = encode_msp2(cmd, payload)
	} else {
		buf = encode_msp(cmd, payload)
	}
	m.write(buf)
}

func MSPInit(dd DevDescription, _usev2 bool) *MSPSerial {
	var fw, api, vers, board, gitrev string
	var m *MSPSerial

	switch dd.klass {
		case DevClass_SERIAL:
		m = NewMSPSerial(dd)
	case DevClass_TCP:
		m = NewMSPTCP(dd)
	case DevClass_UDP:
		m = NewMSPUDP(dd)
  default:
		fmt.Fprintln(os.Stderr, "Unsupported device")
		os.Exit(1)
	}
	m.usev2 = _usev2
	m.Send_msp(msp_API_VERSION, nil)
	_, payload, err := m.Read_cmd(msp_API_VERSION)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read: ", err)
	} else {
		m.vcapi = uint16(payload[1])<<8 |  uint16(payload[2])
		api = fmt.Sprintf("%d.%d", payload[1], payload[2])
	}

	m.Send_msp(msp_FC_VARIANT, nil)
	_, payload, err = m.Read_cmd(msp_FC_VARIANT)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read: ", err)
	} else {
		fw = string(payload[0:4])
	}

	m.Send_msp(msp_FC_VERSION, nil)
	_, payload, err = m.Read_cmd(msp_FC_VERSION)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read: ", err)
	} else {
		vers = fmt.Sprintf("%d.%d.%d", payload[0], payload[1], payload[2])
	}

	m.Send_msp(msp_BUILD_INFO, nil)
	_, payload, err = m.Read_cmd(msp_BUILD_INFO)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read: ", err)
	} else {
		gitrev = string(payload[19:])
	}

	have_name := false
	m.Send_msp(msp_BOARD_INFO, nil)
	_, payload, err = m.Read_cmd(msp_BOARD_INFO)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read: ", err)
	} else {
		if len(payload) > 8 {
 			board = string(payload[9:])
			have_name = true
		} else {
 			board = string(payload[0:4])
		}
	}

	fmt.Fprintf(os.Stderr, "%s v%s %s (%s) API %s", fw, vers, board, gitrev, api)
	if have_name {
		m.Send_msp(msp_NAME, nil)
		_, payload, err = m.Read_cmd(msp_NAME)

		if len(payload) > 0 {
			fmt.Fprintf(os.Stderr, " \"%s\"", payload)
		}
	}
	fmt.Fprintln(os.Stderr, "\n")
	return m
}

func serialise_rx(phase int8, sarm int) ([]byte) {
	buf := make([]byte, 16)
	var aoff = int(0);
	if sarm > 4 && sarm < 9 {
		aoff = (sarm-1)*2;
	}

	binary.LittleEndian.PutUint16(buf[8:10], uint16(1017))
	binary.LittleEndian.PutUint16(buf[10:12], uint16(1442))
	binary.LittleEndian.PutUint16(buf[12:14], uint16(1605))
	binary.LittleEndian.PutUint16(buf[14:16], uint16(1669))
	if aoff != 0 {
		binary.LittleEndian.PutUint16(buf[aoff:aoff+2], uint16(1001))
	}

	switch phase {
	case 0:
		n := rand.Intn(rx_RAND)
		binary.LittleEndian.PutUint16(buf[0:2], uint16(rx_START+n))
		n = rand.Intn(rx_RAND)
		binary.LittleEndian.PutUint16(buf[2:4], uint16(rx_START+n))
		n = rand.Intn(rx_RAND)
		binary.LittleEndian.PutUint16(buf[4:6], uint16(rx_START+n))
		n = rand.Intn(rx_RAND)
		binary.LittleEndian.PutUint16(buf[6:8], uint16(rx_START+n))
	case 1:
		binary.LittleEndian.PutUint16(buf[0:2], uint16(1500))
		binary.LittleEndian.PutUint16(buf[2:4], uint16(1500))
		binary.LittleEndian.PutUint16(buf[4:6], uint16(1500))
		binary.LittleEndian.PutUint16(buf[6:8], uint16(1000))

	case 2:
		binary.LittleEndian.PutUint16(buf[0:2], uint16(1500))
		binary.LittleEndian.PutUint16(buf[2:4], uint16(1500))
		if sarm == 0 {
			binary.LittleEndian.PutUint16(buf[4:6], uint16(2000))
		} else {
			binary.LittleEndian.PutUint16(buf[4:6], uint16(1500))
			binary.LittleEndian.PutUint16(buf[aoff:aoff+2], uint16(2000))
		}
		binary.LittleEndian.PutUint16(buf[6:8], uint16(1000))
	case 3:
		binary.LittleEndian.PutUint16(buf[0:2], uint16(1500))
		binary.LittleEndian.PutUint16(buf[2:4], uint16(1500))
		binary.LittleEndian.PutUint16(buf[4:6], uint16(1500))
		binary.LittleEndian.PutUint16(buf[6:8], uint16(1000))
		if aoff != 0 {
			binary.LittleEndian.PutUint16(buf[aoff:aoff+2], uint16(2000))
		}
	case 4:
		binary.LittleEndian.PutUint16(buf[0:2], uint16(1500))
		binary.LittleEndian.PutUint16(buf[2:4], uint16(1500))
		if sarm == 0 {
			binary.LittleEndian.PutUint16(buf[4:6], uint16(1000))
		} else {
			binary.LittleEndian.PutUint16(buf[4:6], uint16(1500))
			binary.LittleEndian.PutUint16(buf[aoff:aoff+2], uint16(1000))
		}
		binary.LittleEndian.PutUint16(buf[6:8], uint16(1000))
	}
	return buf
}


func deserialise_rx(b []byte) ([]int16) {
	buf := make([]int16, 8)
	for j:= 0; j < 8; j++ {
		n := j*2;
		buf[j] = int16(binary.LittleEndian.Uint16(b[n:n+2]))
	}
	return buf
}

func (m *MSPSerial) test_rx(arm bool, sarm int) () {
	cnt := 0
	var phase = int8(0)

	if sarm != 0 {
		arm = true;
	}

	for ;; {
		if arm {
			if cnt <= 300 || cnt > 1510 {
				phase = 1
			} else if cnt > 1500 {
				phase = 4
			} else if cnt > 310 {
				phase = 3
			} else if cnt > 300 {
				phase = 2
			} else {
				phase = 0
			}
		}
		tdata := serialise_rx(phase, sarm);
		m.Send_msp(msp_SET_RAW_RC, tdata)
		_, _, err := m.Read_cmd(msp_SET_RAW_RC)
		if err == nil {
			m.Send_msp(msp_RC,nil)
			_, payload, err := m.Read_cmd(msp_RC)
			if err == nil {
				txdata := deserialise_rx(tdata)
				fmt.Printf("Tx: %v\n", txdata)
				rxdata := deserialise_rx(payload)
				fmt.Printf("Rx: %v (%05d)", rxdata, cnt)
				var stscmd byte
				if m.vcapi > 0x200 {
					stscmd = msp_STATUS_EX
				} else {
					stscmd = msp_STATUS
				}
				m.Send_msp(stscmd, nil)
				_, payload, err := m.Read_cmd(stscmd)
				if err == nil {
					var status uint32
					status = binary.LittleEndian.Uint32(payload[6:10])
					if status & 1 == 1 {
						fmt.Print(" armed")
					} else {
						if stscmd == msp_STATUS_EX {
							armf := binary.LittleEndian.Uint16(payload[13:15])
							fmt.Printf(" unarmed (%x)", armf)
						} else {
							fmt.Print(" unarmed")
						}
					}
				} else {
					log.Fatalf("MSP_STATUS - %v\n", err)
				}
				switch phase {
				case 0:
					fmt.Printf("\n");
				case 1:
					fmt.Printf(" Quiescent\n");
				case 2:
					fmt.Printf(" Arming\n");
				case 3:
					fmt.Printf(" Min throttle\n");
				case 4:
					fmt.Printf(" Dis-arming\n");
				}
			} else {
				log.Fatalf("MSP_SET_RAW_RC - %v\n", err)
			}
		} else {
			log.Fatalf("MSP_RC - %v\n", err)
		}
		time.Sleep(100 * time.Millisecond)
		cnt++
	}
}
