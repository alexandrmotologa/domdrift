.PHONY: all build build-web build-go test test-canary clean run-studio run-canary

all: build test

build-web:
	cd web && npm install && npm run build
	@mkdir -p internal/server/dist
	@cp -r web/dist/* internal/server/dist/

build-go:
	go build -o bin/domdrift ./cmd/domdrift

build: build-web build-go

test:
	go test -v ./internal/...

test-canary: build-go
	./bin/domdrift extract --url http://127.0.0.1:9090/v1 --schema examples/schemas/ecommerce_product.json --output out/test_v1.jsonl

run-studio: build
	./bin/domdrift studio --port 8282

run-canary: build-go
	./bin/domdrift test-server --port 9090

clean:
	rm -rf bin/ out/ tmp/ selectors.cache.json
