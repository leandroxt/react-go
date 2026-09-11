.PHONY: dev build run check

# Bundler in watch mode alongside the server.
dev:
	@go run ./cmd/build -watch & \
	go run ./cmd/web

# Assets first — they get embedded into the binary.
# -s -w drop the symbol table and DWARF debug info: smaller binary, no loss
# of panic stack traces (those come from the runtime, not DWARF).
build:
	go run ./cmd/build
	go build -ldflags="-s -w" -o bin/app ./cmd/web

run: build
	./bin/app

# Type-checking only; esbuild strips types but never checks them.
check:
	npx tsgo --noEmit
