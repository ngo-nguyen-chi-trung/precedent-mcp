.PHONY: generate mocks build run test e2e cover tidy

SHELL := /bin/bash
MODULE := github.com/ngo/precedent_mcp
UNIT_COVER_DIR := $(CURDIR)/bin/unit
COVER_DIR := $(CURDIR)/bin/cover

generate:
	protoc \
		--go_out=. --go_opt=module=$(MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
		communication/*.proto

mocks:
	go tool mockery

test:
	rm -rf $(UNIT_COVER_DIR) && mkdir -p $(UNIT_COVER_DIR)
	go test -count=1 -coverpkg=./... ./test/... -args -test.gocoverdir=$(UNIT_COVER_DIR)
	go tool covdata percent -i=$(UNIT_COVER_DIR)

e2e:
	go test -tags e2e -count=1 -v ./test/e2e/...

cover: export E2E_COVERDIR := $(COVER_DIR)
cover:
	rm -rf $(COVER_DIR) && mkdir -p $(COVER_DIR)
	go test -tags e2e -count=1 -coverpkg=./... ./test/... -args -test.gocoverdir=$(COVER_DIR)
	go tool covdata percent -i=$(COVER_DIR)
	go tool covdata textfmt -i=$(COVER_DIR) -o=bin/cover.out
	go tool cover -func=bin/cover.out | tail -1

tidy:
	go mod tidy

build:
	go build -o bin/server ./cmd/server
	go build -o bin/gateway ./cmd/gateway

run: build
	./bin/server & server=$$!; \
	./bin/gateway & gateway=$$!; \
	trap 'kill $$server $$gateway 2>/dev/null' EXIT; \
	trap exit INT TERM; \
	wait -n
