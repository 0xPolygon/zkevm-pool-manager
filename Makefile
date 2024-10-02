include version.mk

ARCH := $(shell arch)

ifeq ($(ARCH),x86_64)
	ARCH = amd64
else
	ifeq ($(ARCH),aarch64)
		ARCH = arm64
	endif
endif

GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/dist
GOENVVARS := GOBIN=$(GOBIN) CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH)
GOBINARY := zkevm-pool-manager
GOCMD := $(GOBASE)/cmd

LDFLAGS += -X 'github.com/0xPolygonHermez/zkevm-pool-manager.Version=$(VERSION)'
LDFLAGS += -X 'github.com/0xPolygonHermez/zkevm-pool-manager.GitRev=$(GITREV)'
LDFLAGS += -X 'github.com/0xPolygonHermez/zkevm-pool-mananger.GitBranch=$(GITBRANCH)'
LDFLAGS += -X 'github.com/0xPolygonHermez/zkevm-pool-manager.BuildDate=$(DATE)'

# Check dependencies
# Check for Go
.PHONY: check-go
check-go:
	@which go > /dev/null || (echo "Error: Go is not installed" && exit 1)

# Check for Docker
.PHONY: check-docker
check-docker:
	@which docker > /dev/null || (echo "Error: docker is not installed" && exit 1)

# Check for Docker-compose
.PHONY: check-docker-compose
check-docker-compose:
	@which docker-compose > /dev/null || (echo "Error: docker-compose is not installed" && exit 1)

# Check for Curl
.PHONY: check-curl
check-curl:
	@which curl > /dev/null || (echo "Error: curl is not installed" && exit 1)

# Targets that require the checks
build: check-go
lint: check-go
build-docker: check-docker
build-docker-nc: check-docker
run-rpc: check-docker check-docker-compose
stop: check-docker check-docker-compose
install-linter: check-go check-curl
update-external-dependencies: check-go

.PHONY: build
build: ## Builds the binary locally into ./dist
	$(GOENVVARS) go build -ldflags "all=$(LDFLAGS)" -o $(GOBIN)/$(GOBINARY) $(GOCMD)

.PHONY: build-docker
build-docker: ## Builds a docker image with the pool-manager binary
	docker build -t zkevm-pool-manager -f ./Dockerfile .

.PHONY: build-docker-nc
build-docker-nc: ## Builds a docker image with the pool-manager binary - but without build cache
	docker build --no-cache=true -t zkevm-pool-manager -f ./Dockerfile .

.PHONY: run
run: ## Runs all the components needed to run a local pool-manager
	docker compose up -d cdk-erigon
	docker compose up -d zkevm-pool-db
	sleep 2
	docker compose up -d zkevm-pool-manager

.PHONY: run-db
run-db: ## Runs pool database
	docker compose up -d zkevm-pool-db

.PHONY: stop-db
stop-db: ## Stops pool database
	docker compose down zkevm-pool-db

.PHONY: run-seq
run-seq: ## Runs erigon sequencer
	docker compose up -d cdk-erigon

.PHONY: stop-seq
stop-seq: ## Stops erigon sequencer
	docker compose down cdk-erigon

.PHONY: rm-seq
rm-seq: ## Removes erigon sequencer data
	sudo rm -rf ./data/cdk-erigon
	sudo mkdir ./data/cdk-erigon
	sudo chmod 777 -R ./data/cdk-erigon

.PHONY: stop
stop: ## Stops all services
	docker-compose down

.PHONY: install-linter
install-linter: ## Installs the linter
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2

.PHONY: lint
lint: ## Runs the linter
	export "GOROOT=$$(go env GOROOT)" && $$(go env GOPATH)/bin/golangci-lint run

.PHONY: update-external-dependencies
update-external-dependencies: ## Updates external dependencies like images, test vectors or proto files
	go run ./scripts/cmd/... updatedeps

.PHONY: install-git-hooks
install-git-hooks: ## Moves hook files to the .git/hooks directory
	cp .github/hooks/* .git/hooks

.PHONY: install-mockery
install-mockery: ## Installs mockery with the correct version to generate the mocks
	go install github.com/vektra/mockery/v2@v2.39.0

.PHONY: generate-mocks
generate-mocks: ## Generates mock files
	export "GOROOT=$$(go env GOROOT)" && $$(go env GOPATH)/bin/mockery --name=poolDBInterface --dir=./server --output=./server --outpkg=server --inpackage --structname=poolDBMock --filename=mock_pooldb.go
	export "GOROOT=$$(go env GOROOT)" && $$(go env GOPATH)/bin/mockery --name=senderInterface --dir=./server --output=./server --outpkg=server --inpackage --structname=senderMock --filename=mock_sender.go

.PHONY: test
test: ## Runs test files
	trap '$(STOP)' EXIT; MallocNanoZone=0 go test -count=1 -short -race -p 1 -covermode=atomic -coverprofile=./coverage.out  -coverpkg ./... -timeout 70s ./...

## Help display.
## Pulls comments from beside commands and prints a nicely formatted
## display with the commands and their usage information.
.DEFAULT_GOAL := help

.PHONY: help
help: ## Prints this help
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
	| sort \
	| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
