prefix ?= /usr/local

APP = msp_set_rx
SRC = msp.go msp_set_rx.go
all: $(APP)

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
