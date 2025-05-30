BIN_DIR = bin
CMD_PATHS = $(wildcard cmd/*)
BINARIES=$(foreach x, $(notdir  $(CMD_PATHS)), ${BIN_DIR}/$(x))

.PHONY: bin-dir $(BINARIES)

all:  $(BINARIES)

bin-dir:
	@mkdir -p $(BIN_DIR)

$(BINARIES): | bin-dir
	go build -o $@ ./cmd/$(notdir $@)
