package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	DevClass_NONE = iota
	DevClass_SERIAL
	DevClass_TCP
	DevClass_UDP
	DevClass_BT
)

type DevDescription struct {
	klass  int
	name   string
	param  int
	name1  string
	param1 int
}

var (
	baud   = flag.Int("b", 115200, "Baud rate")
	device = flag.String("d", "", "Serial Device")
	arm    = flag.Bool("a", false, "Arm (take care now)")
	fs     = flag.Bool("fs", false, "Test failsafe")
)

func check_device() DevDescription {
	devdesc := parse_device(*device)
	if devdesc.name == "" {
		for _, v := range []string{"/dev/ttyACM0", "/dev/ttyUSB0"} {
			if _, err := os.Stat(v); err == nil {
				devdesc.klass = DevClass_SERIAL
				devdesc.name = v
				devdesc.param = *baud
				break
			}
		}
	}
	if devdesc.name == "" && devdesc.param == 0 {
		log.Fatalln("No device given\n")
	} else {
		log.Printf("Using device %s\n", devdesc.name)
	}
	return devdesc
}

func resolve_default_gw() string {
	cmds := []string{"ip route show 0.0.0.0/0 | cut -d ' ' -f3",
		"route -n | grep UG | awk '{print $2}'",
		"route -n show  0.0.0.0 | grep gateway | awk '{print $2}'"}

	ostr := os.Getenv("MWP_SERIAL_HOST")
	if ostr != "" {
		return ostr
	}
	for _, c := range cmds {
		out, err := exec.Command("sh", "-c", c).Output()
		ostr := strings.TrimSpace(string(out))
		if err != nil {
			log.Fatal(err)
		} else {
			if len(ostr) > 0 {
				return ostr
			}
		}
	}
	return "__MWP_SERIAL_HOST"
}

func splithost(uhost string) (string, int) {
	port := -1
	host := ""
	if uhost != "" {
		if h, p, err := net.SplitHostPort(uhost); err != nil {
			host = uhost
		} else {
			host = h
			port, _ = strconv.Atoi(p)
		}
	}
	return host, port
}

func parse_device(devstr string) DevDescription {
	dd := DevDescription{name: "", klass: DevClass_NONE}
	if devstr == "" {
		return dd
	}

	if len(devstr) == 17 && (devstr)[2] == ':' && (devstr)[8] == ':' && (devstr)[14] == ':' {
		dd.name = devstr
		dd.klass = DevClass_BT
	} else {
		u, err := url.Parse(devstr)
		if err == nil {
			if u.Scheme == "tcp" {
				dd.klass = DevClass_TCP
			} else if u.Scheme == "udp" {
				dd.klass = DevClass_UDP
			}

			if u.Scheme == "" {
				ss := strings.Split(u.Path, "@")
				dd.klass = DevClass_SERIAL
				dd.name = ss[0]
				if len(ss) > 1 {
					dd.param, _ = strconv.Atoi(ss[1])
				} else {
					dd.param = 115200
				}
			} else {
				if u.RawQuery != "" {
					m, err := url.ParseQuery(u.RawQuery)
					if err == nil {
						if p, ok := m["bind"]; ok {
							dd.param, _ = strconv.Atoi(p[0])
						}
						dd.name1, dd.param1 = splithost(u.Host)
					}
				} else {
					if u.Path != "" {
						parts := strings.Split(u.Path, ":")
						if len(parts) == 2 {
							dd.name1 = parts[0][1:]
							dd.param1, _ = strconv.Atoi(parts[1])
						}
					}
					dd.name, dd.param = splithost(u.Host)
					if dd.name == "__MWP_SERIAL_HOST" {
						dd.name = resolve_default_gw()
					}
				}
			}
		}
	}
	return dd
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of msp_set_rx [options]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	devdesc := check_device()
	s := MSPInit(devdesc)
	if s.swchan == -1 || s.swvalue < 1000 {
		log.Fatalln("Mis-configured arm switch --- see README")
	} else {
		fmt.Printf("Arming set for channel %d / %dus\n", s.swchan+1, s.swvalue)
		s.test_rx(*arm, *fs)
	}
}
