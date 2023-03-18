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
# for ancient firmware
feature RX_MSP

# Modern firmware (e.g inav 1.8 and later)
set receiver_type = MSP
```

Note: `MSP_SET_RAW_RC` assumes a "AERT" channel map. `msp_set_rx` uses the map set on the FC to send data, but always reports "AERT". Note that earlier versions of `msp_set_rx` assumed "AERT" unless you explicitly told it to use a different map (the now removed `-m` option).

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
  -2	Use MSPv2
  -A int
    	Arm Switch, (5-16), assumes 2000us will arm
  -a	Arm (take care now) [only inav versions supporting stick arming]
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
# Arm on switch 10 (set range as 1800-2100 in CLI/configurator)
$ ./msp_set_rx -A 10 -d /dev/ttyACM0
# MSPv2
$ ./msp_set_rx -2 -A 10 -d /dev/ttyACM0
```

Note: On Linux, `/dev/ttyUSB0` and `/dev/ttyACM0` are automatically detected.

### Arm / Disarm test

The application can also test arm / disarm, with the `-a` option (where the inav versions supporting stick arming) (or `-A n` for switch arming). In this mode, the application:

* Sets a quiescent state for 30 seconds
* Arms using the configured  (stick or switch) command
* Maintains low-throttle (< 1300uS) for two minutes
* Disarms (stick or switch command)

**The vehicle must be in a state that will allow arming: [inav wiki article](https://github.com/iNavFlight/inav/wiki/%22Something%22-is-disabled----Reasons).**

If `nav_extra_arming_safety = ALLOW_BYPASS` is set on the FC, it will be honoured to allow arming with bypass.

Summary of output (`##` indicates a comment, repeated lines removed). 16 channels are reported (earlier versions displayed 8).

Post 2020-01-11, shows armed status and iteration count, switch arming:

```
./msp_set_rx -A 5
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
Note that from 2020-04-22, in the disarmed state the arming status is displayed as a hex number, now using 32bit MSP2_INAV_STATUS where possible (after inav 1.8.1).:

```
... unarmed (0) Quiescent
... unarmed (800) Quiescent ## => Navigation Unsafe
```

Real example ....

```
$ ./msp_set_rx -A 5
2020/04/22 16:42:21 Using device /dev/ttyACM0
INAV v2.5.0 WINGFC (a70ac039) API 2.4 "MrFloaty"

Tx: [1500 1500 1500 1000 1001 1442 1605 1669]
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (00000) unarmed (40200) Quiescent
Tx: [1500 1500 1500 1000 1001 1442 1605 1669]
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (00001) unarmed (40200) Quiescent
Tx: [1500 1500 1500 1000 1001 1442 1605 1669]
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (00002) unarmed (40200) Quiescent
Tx: [1500 1500 1500 1000 1001 1442 1605 1669]
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (00003) unarmed (40200) Quiescent
Tx: [1500 1500 1500 1000 1001 1442 1605 1669]
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (00004) unarmed (40200) Quiescent
Tx: [1500 1500 1500 1000 1001 1442 1605 1669]
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (00005) unarmed (200) Quiescent
Tx: [1500 1500 1500 1000 1001 1442 1605 1669]
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (00006) unarmed (0) Quiescent

```

| value | reason |
| ---- | ---- |
| 0x40000 | No RX link |
| 0x200 | calibrating |

which seems reasonable, RX not recognised until sensor calibration complete.

Older version, stick arming:

```
$ ./msp_set_rx -d /dev/ttyACM0 -a
2018/11/13 18:47:15 Using device /dev/ttyACM0
INAV v2.1.0 MATEKF405 (f740c47c) API 2.2 "big-quad"
## for the first 30 seconds
Tx: [1500 1500 1500 1000 1017 1442 1605 1669]
Rx: [1500 1500 1500 1000 1017 1442 1605 1669] Quiescent
...
## for 30 - 31 seconds
Tx: [1500 1500 2000 1000 1017 1442 1605 1669]
Rx: [1500 1500 2000 1000 1017 1442 1605 1669] Arming
...
## for the next two minutes
Tx: [1500 1500 1500 1000 1017 1442 1605 1669]
Rx: [1500 1500 1500 1000 1017 1442 1605 1669] Min throttle
...
## After 2 minutes & 30 seconds
Tx: [1500 1500 1000 1000 1017 1442 1605 1669]
Rx: [1500 1500 1000 1000 1017 1442 1605 1669] Dis-arming
...
## After 2 minutes & 31 seconds
Tx: [1500 1500 1500 1000 1017 1442 1605 1669]
Rx: [1500 1500 1500 1000 1017 1442 1605 1669] Quiescent
```

Even ancient F1 (patched for RX_MSP)

```
$ ./msp_set_rx -A 5
2020/01/12 11:00:01 Using device /dev/ttyACM0
INAV v1.7.3 CC3D (772e015f) API 2.0
...
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (00300) unarmed Quiescent
Tx: [1500 1500 1500 1000 2000 1442 1605 1669]
Rx: [1500 1500 1500 1000 2000 1442 1605 1669] (00301) unarmed Arming
Tx: [1500 1500 1500 1000 2000 1442 1605 1669]
Rx: [1500 1500 1500 1000 2000 1442 1605 1669] (00302) armed Arming
...
Tx: [1500 1500 1500 1000 2000 1442 1605 1669]
Rx: [1500 1500 1500 1000 2000 1442 1605 1669] (00311) armed Min throttle
...
Rx: [1500 1500 1500 1000 2000 1442 1605 1669] (01500) armed Min throttle
Tx: [1500 1500 1500 1000 1000 1442 1605 1669]
Rx: [1500 1500 1500 1000 1000 1442 1605 1669] (01501) armed Dis-arming
Tx: [1500 1500 1500 1000 1000 1442 1605 1669]
Rx: [1500 1500 1500 1000 1000 1442 1605 1669] (01502) armed Dis-arming
Tx: [1500 1500 1500 1000 1000 1442 1605 1669]
Rx: [1500 1500 1500 1000 1000 1442 1605 1669] (01503) unarmed Dis-arming
...
Tx: [1500 1500 1500 1000 1001 1442 1605 1669]
Rx: [1500 1500 1500 1000 1001 1442 1605 1669] (01511) unarmed Quiescent
```

While this attempts to arm at a safe throttle value, removing props or using a current limiter is recommended.

### Failsafe test

If the failsafe mode is commanded (`-fs`), then no RC data is sent between 40s and 50s. This will cause the FS to enter failsafe for this period.

## Caveats

* Ensure you provide (at least) 5Hz RX data, but don't overload the FC; MSP is a request-response protocol, don't just "spam" the FC via a high frequency timer and ignore the responses.
* Ensure you're set the correct AUX range to arm
* Ensure you've met the required arming conditions
* For F1, no other `RX_xxx` feature set.
* Use a supported FC
* Remove the props etc.

## Licence

Whatever approximates to none / public domain in your locale. 0BSD (Zero clause BSD)  if an actual license is required by law.
