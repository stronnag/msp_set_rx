prefix ?= /usr/local

APP = msp_set_rx
SRC = msp.go msp_set_rx.go
all: $(APP)

$(APP): $(SRC) go.sum
	go build -ldflags "-w -s" -o $@

go.sum: go.mod
	go mod tidy

clean:
	go clean

install: $(APP)
	-install -d $(prefix)/bin
	-install -s $(APP) $(prefix)/bin/$(APP)
