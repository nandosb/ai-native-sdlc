ifneq (,$(wildcard .env))
  include .env
  export
endif

BINARY  := sdlc
WEB_DIR := web

.PHONY: all build web server run dev vet tidy clean

## all: build frontend + backend
all: web build

## build: compile Go binary (no CGO)
build:
	CGO_ENABLED=0 go build -o $(BINARY) ./cmd/sdlc/

## web: install deps + build React frontend
web:
	cd $(WEB_DIR) && npm install && npm run build

## server: build and start the server
server: build
	./$(BINARY)

## run: build everything and start
run: all
	./$(BINARY)

## dev: start frontend dev server (hot reload, proxies to :3000)
dev:
	cd $(WEB_DIR) && npm run dev

## vet: run Go static checks
vet:
	go vet ./...

## tsc: type-check frontend
tsc:
	cd $(WEB_DIR) && npx tsc -b

## tidy: clean up go.mod
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf $(WEB_DIR)/dist
