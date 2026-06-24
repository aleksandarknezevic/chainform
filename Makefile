BINARY := chainform
PKG := github.com/aleksandarknezevic/chainform
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: all build install test vet fmt tidy run-plan run-mainnet-show run-mainnet-plan clean

all: vet test build

build: ## Build the chainform binary into ./bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/chainform

install: ## Install chainform into $GOBIN
	go install -ldflags "$(LDFLAGS)" ./cmd/chainform

test: ## Run the test suite
	go test ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format the codebase
	gofmt -w .

tidy: ## Tidy go.mod / go.sum
	go mod tidy

run-plan: build ## Run `plan` against the example config using the offline demo reader
	./bin/$(BINARY) plan -f examples/protocol.hcl --mock

run-mainnet-show: build ## Inspect Lido + Chainlink on Ethereum mainnet (needs RPC_URL)
	./bin/$(BINARY) show -f examples/mainnet.hcl

run-mainnet-plan: build ## Plan against mainnet example invariants (needs RPC_URL)
	./bin/$(BINARY) plan -f examples/mainnet.hcl

clean:
	rm -rf bin
