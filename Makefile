CMDS := $(notdir $(basename $(wildcard cmd/*)))
CMD_TARGETS := $(addprefix bin/, $(CMDS))

default: build

build: $(CMD_TARGETS)

docker:
	docker build -f docker/stellar-crawler.dockerfile -t tinco/stellar-crawler .
	docker push tinco/stellar-crawler

.PHONY: $(CMD_TARGETS) docker
$(CMD_TARGETS):
	go build -o bin/$(notdir $(basename $@)) cmd/$(notdir $(basename $@))/*
