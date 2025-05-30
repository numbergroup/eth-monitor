BIN_DIR = bin
CMD_PATHS = $(wildcard cmd/*)
BINARIES=$(foreach x, $(notdir  $(CMD_PATHS)), ${BIN_DIR}/$(x))

.PHONY: bin-dir $(BINARIES) static-linux all

all:  $(BINARIES)

bin-dir:
	@mkdir -p $(BIN_DIR)

$(BINARIES): | bin-dir
	go build -o $@ ./cmd/$(notdir $@)

static-linux: bin-dir
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build -ldflags '-extldflags "-static"' -tags netgo,osusergo -o ${BIN_DIR}/monitor ./cmd/monitor
