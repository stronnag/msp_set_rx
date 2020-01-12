# MSP SET_RC considered dangerous

## Overview

This golang program exercises `MSP SET_RAW_RC`.

### Why

Every few months, someone will come along on iNav github / RC Groups / Telegram / some other random support channel and state that RX_MSP doesn't work.

Well it does, if you do it right. This example demonstrates usage.

As Go is available on pretty much any OS, you can easily verify that it works.

## FC Prerequisites

A supported FC

```
# for ancient firmware
map AERT5678
feature RX_MSP

# Modern firmware (e.g iNav 1.8 and later)
map AERT
set receiver_type = MSP
```

Update RX data at 5Hz or better.

Consider also (post iNav 2.1) custom firmware with `#define USE_MSP_RC_OVERRIDE` in `target/common.h` and enabling the MSP RC override flight mode.

## Building

* Clone this repository
* If you haven't previously go got `tarm/serial` then do that:

 ```
 go get github.com/tarm/serial
 ```

* Build the test application

 ```
 go build
 ```

This should result in a `msp_set_rx` application.

## Usage

```
$ ./msp_set_rx --help
Usage of msp_set_rx [options]
  -2	Use MSPv2
  -A int
    	Arm Switch, (5-8), assumes 2000us will arm
  -a	Arm (take care now) [only iNav versions supporting stick arming]
  -b int
    	Baud rate (default 115200)
  -d string
    	Serial Device
```

Sets random (but safe) values:

```
$ ./msp_set_rx -d /dev/ttyUSB0 [-b baud]
# and hence, probably, for example
C:\> msp_set_rx.exe -d COM42 -b 115200
# Arm on switch 5 (set range as 1800-2100 in CLI/configurator)
$ ./msp_set_rx -A 5 -d /dev/ttyACM0
# MSPv2
$ ./msp_set_rx -2 -A 5 -d /dev/ttyACM0
```

Note: On Linux, `/dev/ttyUSB0` and `/dev/ttyACM0` are automatically detected.

### Arm / Disarm test

The application can also test arm / disarm, with the `-a` option (where the iNav versions supporting stick arming) (or `-A n` for switch arming). In this mode, the application:

* Sets a quiescent state for 30 seconds
* Arms using the configured  (stick or switch) command
* Maintains min-throttle (1000uS) for two minutes
* Disarms (stick or switch command)

The vehicle must be in a state that will allow arming: [iNav wiki article](https://github.com/iNavFlight/inav/wiki/%22Something%22-is-disabled----Reasons).

Summary of output (`##` indicates a comment, repeated lines removed).

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

## Caveats

* Ensure you provide (at least) 5Hz RX data, but don't overload the FC; MSP is a request-response protocol, don't just "spam" the FC via a high frequency timer and the ignore responses.
* Esnure you're set the correct AUX range to arm
* Ensure you've met the required arming conditions
* Correct `map` for you FC version (4 character, 8 character etc.)
* For F1, no other `RX_xxx` feature set.
* Use a supported FC

## Licence

Whatever approximates to none / public domain in your locale.
