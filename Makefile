.PHONY: build test vet generate run-server run-client

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

# Regenerate the calculator wrapper. Requires the Cap'n Proto bindings
# (calculator.capnp.go) to already exist; see docs for `capnp compile`.
generate:
	go run ./cmd/capwrap-gen examples/calculator/calc/calculator.capnp

run-server:
	go run ./examples/calculator/server

run-client:
	go run ./examples/calculator/client -name Huy -a 123 -b 456
