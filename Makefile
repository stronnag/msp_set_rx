prefix ?= $$HOME/.local

APP = msp_set_rx
UNAME := $(shell uname)

BTSRC = btaddr_other.go
ifeq ($(UNAME), Linux)
 BTSRC = btaddr_linux.go
endif

SRC = msp.go msp_set_rx.go $(BTSRC) inav_misc.go event_loop.go

all: $(APP) arm_status

$(APP): $(SRC) go.sum
	go build -ldflags "-w -s" -o $@ $(SRC)

go.sum: go.mod
	go mod tidy

arm_status: arm_status.go
	go build -ldflags "-w -s" -o $@ $<

clean:
	go clean

install: $(APP)
	-install -d $(prefix)/bin
	-install -s $(APP) $(prefix)/bin/$(APP)
