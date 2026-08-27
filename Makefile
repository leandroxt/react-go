.PHONY: dev build run check

# Bundler in watch mode alongside the server.
dev:
	@go run ./cmd/build -watch & \
	go run ./cmd/web

# Assets first — they get embedded into the binary.
build:
	go run ./cmd/build
	go build -o bin/app ./cmd/web

run: build
	./bin/app

# Type-checking only; esbuild strips types but never checks them.
check:
	npx tsgo --noEmit
