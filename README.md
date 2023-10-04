# MSP_SET_RAW RC considered harmful

## Overview

This golang program exercises `MSP SET_RAW_RC`.

### Why

Every few months, someone will come along on INAV Github / RC Groups / Telegram / Discord / some other random support channel and state that `RX_MSP` / `set receiver_type = MSP`  doesn't work.

Well it does, if you do it right. This example demonstrates usage.

As Go is available on pretty much any OS, you can easily verify that it works, with this small application.

## FC Prerequisites

A supported FC

```
# Modern firmware (e.g INAV 1.8 and later)
set receiver_type = MSP
```

For older versions of this example, with INAV prior to INAV 1.8:

```
# for ancient firmware
feature RX_MSP
```

Note: `MSP_RC` assumes a "AERT" channel map. `MSP_SET_RAW_RC` honours the FC's channel map, so `msp_set_rx` uses the map set on the FC to send data, but always reports "AERT".

Update RX data at 5Hz or better.

Consider also (post inav 2.1) custom firmware with `#define USE_MSP_RC_OVERRIDE` in `target/common.h` and enabling the MSP RC override flight mode. It is also advisable to `make <TARGET_clean>` when changing such defines.

Note that as this tool can cause motors to run, the usual "don't be stupid / remove props / secure the vehicle" warnings apply.

## Building

* Clone this repository
* Build the test application

 ```
 make
 ```

This should result in a `msp_set_rx` application.

## Usage

```
$ ./msp_set_rx --help
Usage of msp_set_rx [options]
  -a	Arm (take care now)
  -b int
    	Baud rate (default 115200)
  -d string
    	Serial Device
  -fs
    	Test failsafe
```

Sets random (but safe-ish) values:

```
$ ./msp_set_rx -d /dev/ttyUSB0 [-b baud]
# and hence, probably, for example
C:\> msp_set_rx.exe -d COM42 -b 115200
# Arm on switch set in CLI/configurator)
$ ./msp_set_rx -a -d /dev/ttyACM0
```

Note: On Linux, `/dev/ttyUSB0` and `/dev/ttyACM0` are automatically detected.

### Arm / Disarm test

The application can also test arm / disarm, with the `-a` option. In this mode, the application:

* Sets a quiescent state for 30 seconds
* Arms using the configured switch
* Maintains low-throttle (< 1300uS) for two minutes
* Disarms (switch)

**The vehicle must be in a state that will allow arming: [inav wiki article](https://github.com/iNavFlight/inav/wiki/%22Something%22-is-disabled----Reasons).**

If `nav_extra_arming_safety = ALLOW_BYPASS` is set on the FC, it will be honoured to allow arming with bypass.

Summary of output (`##` indicates a comment, repeated lines removed). 16 channels are reported (earlier versions displayed 8).

Post 2020-01-11, shows armed status and iteration count, switch arming:

```
./msp_set_rx -a
2020/01/12 10:13:46 Using device /dev/ttyACM0
INAV v2.4.0 SPRACINGF3EVO (fa4e2426) API 2.4 "Evotest-V"
## for the first 30 seconds
Tx: [1500 1500 1500 1000 1001 1442 1605 1669]
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (00000) unarmed Quiescent
...
## for 30 - 31 seconds
Tx: [1500 1500 1500 1000 2000 1442 1605 1669]
Rx: [1500 1500 1500 1000 2000 1442 1605 1669] (00301) unarmed Arming
Tx: [1500 1500 1500 1000 2000 1442 1605 1669]
Rx: [1500 1500 1500 1000 2000 1442 1605 1669] (00302) armed Arming
...
## for the next two minutes
Tx: [1500 1500 1500 1000 2000 1442 1605 1669]
Rx: [1500 1500 1500 1000 2000 1442 1605 1669] (00311) armed Min throttle
...
## After 2 minutes & 30 seconds
Tx: [1500 1500 1500 1000 1000 1442 1605 1669]
Rx: [1500 1500 1500 1000 1000 1442 1605 1669] (01501) armed Dis-arming
Tx: [1500 1500 1500 1000 1000 1442 1605 1669]
Rx: [1500 1500 1500 1000 1000 1442 1605 1669] (01502) armed Dis-arming
Tx: [1500 1500 1500 1000 1000 1442 1605 1669]
Rx: [1500 1500 1500 1000 1000 1442 1605 1669] (01503) unarmed Dis-arming
## After 2 minutes & 31 seconds
Tx: [1500 1500 1500 1000 1001 1442 1605 1669]
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (01511) unarmed Quiescent
```
Note that from 2020-04-22, in the disarmed state the arming status is displayed as a hex number, now using 32bit `MSP2_INAV_STATUS` where possible (after inav 1.8.1).:

Real example .... using the INAV_SITL. Note that we send `AETR` (`MSP_SET_RAW_RC`) but checking with `MSP_RC` reports `AERT`.

* Repetitive bits trimmed
* 18 channels supported by `MSP_SET_RAW_RC`
* Timers are used for arming / disarming. It is would also be possible to reduce the time in these transitions by using the status from `MSP2_INAV_STATUS`.

```
$ msp_set_rx -a -d tcp://localhost:5761
2023/05/21 11:34:57 Using device localhost
INAV v7.0.0 SITL (ca2202ea) API 2.5
nav_extra_arming_safety: 1 (bypass true)
 map AETR "BENCHYMCTESTY"
BOX: ARM;PREARM;ANGLE;HORIZON;TURN ASSIST;HEADING HOLD;CAMSTAB;NAV POSHOLD;LOITER CHANGE;NAV RTH;NAV WP;HOME RESET;GCS NAV;WP PLANNER;MISSION CHANGE;NAV CRUISE;NAV COURSE HOLD;SOARING;NAV ALTHOLD;MANUAL;NAV LAUNCH;SERVO AUTOTRIM;AUTO TUNE;AUTO LEVEL TRIM;BEEPER;BEEPER MUTE;OSD OFF;BLACKBOX;KILLSWITCH;FAILSAFE;CAMERA CONTROL 1;CAMERA CONTROL 2;CAMERA CONTROL 3;OSD ALT 1;OSD ALT 2;OSD ALT 3;
Tx: [1500 1500 1000 2000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 2000 1000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000] (00000) unarmed (40200) Quiescent
...
Tx: [1500 1500 1000 2000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 2000 1000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000] (00003) unarmed (40200) Quiescent
Tx: [1500 1500 1000 2000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 2000 1000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000] (00004) unarmed (40000) Quiescent
Tx: [1500 1500 1000 2000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 2000 1000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000] (00005) unarmed (0) Quiescent
...
Tx: [1500 1500 1000 2000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 2000 1000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000] (00300) unarmed (0) Quiescent
Tx: [1500 1500 1000 1999 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1999 1000 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000] (00301) unarmed (0) Arming
Tx: [1500 1500 1000 1999 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1999 1000 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000] (00302) armed Arming
...
Tx: [1500 1500 1000 1999 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1999 1000 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000] (00310) armed Arming
Tx: [1500 1500 1189 1500 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1500 1189 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000] (00311) armed Low throttle
Tx: [1500 1500 1299 1500 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1500 1299 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000] (00312) armed Low throttle
...
Tx: [1500 1500 1284 1500 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1500 1284 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000] (01500) armed Low throttle
Tx: [1500 1500 1000 1500 1000 1000 1000 1000 1000 999 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1500 1000 1000 1000 1000 1000 1000 999 1000 1000 1000 1000 1000 1000 1000 1000] (01501) armed Dis-arming
Tx: [1500 1500 1000 1500 1000 1000 1000 1000 1000 999 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1500 1000 1000 1000 1000 1000 1000 999 1000 1000 1000 1000 1000 1000 1000 1000] (01502) armed Dis-arming
Tx: [1500 1500 1000 1500 1000 1000 1000 1000 1000 999 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1500 1000 1000 1000 1000 1000 1000 999 1000 1000 1000 1000 1000 1000 1000 1000] (01503) unarmed (8) Dis-arming
...
Tx: [1500 1500 1000 1500 1000 1000 1000 1000 1000 999 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1500 1000 1000 1000 1000 1000 1000 999 1000 1000 1000 1000 1000 1000 1000 1000] (01509) unarmed (8) Dis-arming
Tx: [1500 1500 1000 1500 1000 1000 1000 1000 1000 999 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1500 1000 1000 1000 1000 1000 1000 999 1000 1000 1000 1000 1000 1000 1000 1000] (01510) unarmed (8) Dis-arming
Tx: [1500 1500 1000 2000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 2000 1000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000] (01511) unarmed (8) Quiescent
Tx: [1500 1500 1000 2000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 2000 1000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000] (01512) unarmed (8) Quiescent
```

| value | reason |
| ---- | ---- |
| 0x40000 | No RX link |
| 0x200 | calibrating |

which seems reasonable, RX not recognised until sensor calibration complete.

While this tool attempts to arm at a safe throttle value, removing props or using a current limiter is recommended. Using the [INAV_SITL](https://github.com/iNavFlight/inav/blob/master/docs/SITL/SITL.md) is also a good option. A suitable configuration for such experiments is described in the [fl2sitl wiki](https://github.com/stronnag/bbl2kml/wiki/fl2sitl#sitl-configuration)

### Data shown

```
Tx: [1500 1500 1000 2000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 2000 1000 1000 1000 1000 1000 1000 1001 1000 1000 1000 1000 1000 1000 1000 1000] (00003) unarmed (40200) Quiescent
...
Tx: [1500 1500 1189 1500 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000]
Rx: [1500 1500 1500 1189 1000 1000 1000 1000 1000 1800 1000 1000 1000 1000 1000 1000 1000 1000] (00311) armed Low throttle
```
Pairs of transmitted `Tx:` and received `Rx:` data (`MSP_SET_RAW_RC` / `MSP_RC`). First four channels are,  for `Tx:` are according to the configured `map`, and for `Rx:` `AERT`. These are followed by followed by channels 5-18.

The `Rx:` line also shows (first stanza) application timer (`00003` ((deciseconds)), arm state (`unarmed`), arming flags (`(40200`) and application mode (`Quiescent`). Where is arm state is non-blocking, the numeric value is not shown (last line).

### Failsafe test

If the failsafe mode is commanded (`-fs`), then no RC data is sent between 40s and 50s. This will cause the FS to enter failsafe for this period.

## Other examples

The [flightlog2kml](https://github.com/stronnag/bbl2kml) project contains a tool [fl2sitl](https://github.com/stronnag/bbl2kml/wiki/fl2sitl) that replays a blackbox log using the [INAV SITL](https://github.com/iNavFlight/inav/blob/master/docs/SITL/SITL.md). Specifically, this uses MSP and MSP_SET_RAW_RC to establish vehicle characteristics, monitor the vehicle status, arm the vehicle and set RC values for AETR and switches during log replay simulation to effectively "fly" the SITL for the recorded flight.

The MSP initialisation, MSP status monitoring and MSP RC management code is in [msp.go](https://github.com/stronnag/bbl2kml/blob/master/pkg/sitlgen/msp.go), specifically the `init()` and `run()` functions. Arming / disarming in [sitlgen.go](https://github.com/stronnag/bbl2kml/blob/master/pkg/sitlgen/sitlgen.go), `arm_action()` function.

This is more comprehensive (and complex) example.

## Caveats

* Ensure you provide (at least) 5Hz RX data, but don't overload the FC; MSP is a request-response protocol, don't just "spam" the FC via a high frequency timer and ignore the responses.
* Ensure you're set the correct AUX range to arm
* Ensure you've met the required arming conditions
* Use a supported FC or the SITL
* Remove the props etc.

## Licence

Whatever approximates to none / public domain in your locale. 0BSD (Zero clause BSD)  if an actual license is required by law.
