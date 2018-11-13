# MSP SET_RC considered dangerous

## Overview

This golang program exercises `MSP SET_RAW_RC`. 

### Why

Every few months, someone will come along on iNav github / RC Groups / Slack / some other random support channel and state that RX_MSP doesn't work. 

Well it does, if you do it right. This example demonstrates usage.

## Prerequisites
```
map AERT

# for ancient firmware
feature RX_MSP              

# iNav 1.8 and later
set receiver_type = MSP     
```

## Building

* Clone this repostitory
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
  -a	Arm (take care now)
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
```

### Arm / Disarm test

The application can also test arm / disarm, with the `-a` option. In this mode, the application:

* Sets a quiescent state for 30 seconds
* Arms using the customary stick command
* Maintains min-throttle for two minutes
* Disarms (stick command)

Summary of output (`##` indicates a comment, repeated lines removed).

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

While this attempts to arm at a safe throttle value, removing props or using a current limiter is recommended.


## Licence

Whatever approximates to none / public domain in your locale.
