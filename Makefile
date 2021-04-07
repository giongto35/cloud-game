# Makefile includes some useful commands to build or format incentives
# More commands could be added

# Variables
PROJECT = cloud-game
REPO_ROOT = github.com/giongto35
ROOT = ${REPO_ROOT}/${PROJECT}

fmt:
	@goimports -w cmd pkg tests
	@gofmt -s -w cmd pkg tests

compile: fmt
	@go install ./cmd/...

check: fmt
	@golangci-lint run cmd/... pkg/...
#	@staticcheck -checks="all,-S1*" ./cmd/... ./pkg/... ./tests/...

dep:
	go mod download
#	go mod tidy

# NOTE: there is problem with go mod vendor when it delete github.com/gen2brain/x264-go/x264c causing unable to build. https://github.com/golang/go/issues/26366
#build.cross: build
#	CGO_ENABLED=1 GOOS=darwin GOARC=amd64 go build --ldflags '-linkmode external -extldflags "-static"' -o bin/coordinator-darwin ./cmd/coordinator
#	CGO_ENABLED=1 GOOS=darwin GOARC=amd64 go build --ldflags '-linkmode external -extldflags "-static"' -o bin/worker-darwin ./cmd/worker
#	CC=arm-linux-musleabihf-gcc GOOS=linux GOARC=amd64 CGO_ENABLED=1 go build --ldflags '-linkmode external -extldflags "-static"' -o bin/coordinator-linu ./cmd/coordinator
#	CC=arm-linux-musleabihf-gcc GOOS=linux GOARC=amd64 CGO_ENABLED=1 go build --ldflags '-linkmode external -extldflags "-static"' -o bin/worker-linux ./cmd/worker

# A user can invoke tests in different ways:
#  - make test runs all tests;
#  - make test TEST_TIMEOUT=10 runs all tests with a timeout of 10 seconds;
#  - make test TEST_PKG=./model/... only runs tests for the model package;
#  - make test TEST_ARGS="-v -short" runs tests with the specified arguments;
#  - make test-race runs tests with race detector enabled.
TEST_TIMEOUT = 60
TEST_PKGS ?= ./cmd/... ./pkg/...
TEST_TARGETS := test-short test-verbose test-race test-cover
.PHONY: $(TEST_TARGETS) test tests
test-short:   TEST_ARGS=-short
test-verbose: TEST_ARGS=-v
test-race:    TEST_ARGS=-race
test-cover:   TEST_ARGS=-cover
$(TEST_TARGETS): test

test: compile
	@go test -timeout $(TEST_TIMEOUT)s $(TEST_ARGS) $(TEST_PKGS)

test-e2e: compile
	@go test ./tests/e2e/...

cover:
	@go test -v -covermode=count -coverprofile=coverage.out $(TEST_PKGS)
#	@$(GOPATH)/bin/goveralls -coverprofile=coverage.out -service=travis-ci -repotoken $(COVERALLS_TOKEN)

clean:
	@rm -rf bin
	@rm -rf build
	@go clean ./cmd/*

build:
	CGO_ENABLED=0 go build -ldflags '-w -s' -o bin/coordinator$(EXT) ./cmd/coordinator
	go build -buildmode=exe -tags static -ldflags '-w -s' $(EXT_WFLAGS) -o bin/worker$(EXT) ./cmd/worker

verify-cores:
	go test -run TestAllEmulatorRooms ./pkg/worker/room -v -renderFrames $(GL_CTX) -outputPath "../../../_rendered"

dev.build: compile build

dev.build-local:
	CGO_ENABLED=0 go build -o bin/coordinator ./cmd/coordinator
	go build -buildmode=exe -o bin/worker ./cmd/worker

dev.run: dev.build-local
	./bin/coordinator --v=5 &
	./bin/worker --coordinatorhost localhost:8000

dev.run-docker:
	docker rm cloud-game-local -f || true
	CLOUD_GAME_GAMES_PATH=$(PWD)/assets/games docker-compose up --build

# RELEASE
# Builds the app for new release.
#
# Folder structure:
#   - assets/
#   	- games/ (shared between both executables)
#   	- cores/ (filtered by extension)
#   - web/
#   - coordinator
#   - worker
#   - config.yaml (shared)
#
# Config params:
# - RELEASE_DIR: the name of the output folder (default: release).
# - CONFIG_DIR: search dir for core config files.
# - DLIB_TOOL: the name of a dynamic lib copy tool (with params) (e.g., ldd -x -y; defalut: ldd).
# - DLIB_SEARCH_PATTERN: a grep filter of the output of the DLIB_TOOL (e.g., mylib.so; default: .*so).
#   Be aware that this search pattern will return only matched regular expression part and not the whole line.
#   de. -> abc def ghj -> def
#   Makefile special symbols should be escaped with \.
# - DLIB_ALTER: a special flag to use altered dynamic copy lib tool for macOS only.
# - CORE_EXT: a glob pattern to filter the cores that are copied into the release.
# - CFG_EXT: a glob pattern to copy config file into the release (default: *.cfg).
#
# Example:
#   make release DLIB_TOOL="ldd -x" DLIB_SEARCH_PATTERN=/usr/lib.*\\\\s CORE_EXT=*.so
#
RELEASE_DIR ?= release
CONFIG_DIR = configs
DLIB_TOOL ?= ldd
DLIB_SEARCH_PATTERN ?= .*so
DLIB_ALTER ?= false
CORE_EXT ?= *_libretro.so
CFG_EXT ?= *.cfg
COORDINATOR_DIR = ./$(RELEASE_DIR)
WORKER_DIR = ./$(RELEASE_DIR)
CORES_DIR = assets/cores
GAMES_DIR = assets/games
.PHONY: release
.SILENT: release
release: clean build
	rm -rf ./$(RELEASE_DIR) && mkdir ./$(RELEASE_DIR)
	mkdir -p $(COORDINATOR_DIR) && mkdir -p $(WORKER_DIR)
	cp ./bin/coordinator $(COORDINATOR_DIR) && cp ./bin/worker $(WORKER_DIR)
	chmod +x $(COORDINATOR_DIR)/coordinator $(WORKER_DIR)/worker
    ifeq ($(DLIB_ALTER),false)
		for bin in $$($(DLIB_TOOL) $(WORKER_DIR)/worker | grep -oE $(DLIB_SEARCH_PATTERN)); \
			do cp -v "$$bin" $(WORKER_DIR); \
		done
    else
		$(DLIB_TOOL) $(WORKER_DIR) $(WORKER_DIR)/worker
    endif
	cp -R ./web $(COORDINATOR_DIR)
	mkdir -p $(WORKER_DIR)/$(GAMES_DIR)
    ifneq (,$(wildcard ./$(GAMES_DIR)))
		cp -R ./$(GAMES_DIR) $(WORKER_DIR)/assets
    endif
	mkdir -p $(WORKER_DIR)/$(CORES_DIR)
	cp ./$(CORES_DIR)/$(CFG_EXT) $(WORKER_DIR)/$(CORES_DIR)
    ifneq (,$(wildcard ./$(CORES_DIR)/$(CORE_EXT)))
		cp -R ./$(CORES_DIR)/$(CORE_EXT) $(WORKER_DIR)/$(CORES_DIR)
    endif
	cp ./$(CONFIG_DIR)/config.yaml ./$(RELEASE_DIR)
