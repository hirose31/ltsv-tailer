.PHONY: help
.DEFAULT_GOAL := help

branch := $(shell git rev-parse --abbrev-ref HEAD)
version := $(shell git describe --tags --always --dirty)
revision := $(shell git rev-parse HEAD)
release := $(shell git describe --tags 2>/dev/null | cut -d"-" -f 1,2)

GO_LDFLAGS := "-X main.Branch=${branch} -X main.Version=${version} -X main.Revision=${revision}"

help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

ltsv-tailer: cmd/ltsv-tailer/main.go pkg/*/*.go ## build ltsv-tailer
	go build -ldflags $(GO_LDFLAGS) -o $@ $<

ltsv-tailer-debug: cmd/ltsv-tailer/main.go pkg/*/*.go ## build ltsv-tailer-debug
	go build -ldflags $(GO_LDFLAGS) -race -o $@ $<

clean: ## clean
	$(RM) ltsv-tailer ltsv-tailer-debug
