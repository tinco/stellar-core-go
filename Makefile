CMDS := $(notdir $(basename $(wildcard cmd/*)))
CMD_TARGETS := $(addprefix bin/, $(CMDS))

default: build

build: $(CMD_TARGETS)

.PHONY: $(CMD_TARGETS)
$(CMD_TARGETS):
	go build -o bin/$(notdir $(basename $@)) cmd/$(notdir $(basename $@))/*
