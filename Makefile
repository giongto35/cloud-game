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
	go build -a -tags netgo -ldflags '-w' -o bin/coordinator ./cmd/coordinator
	go build -a -tags netgo -ldflags '-w' -o bin/worker ./cmd/worker

dev.tools:
	./hack/scripts/install_tools.sh

dev.build: compile build

dev.build-local:
	go build -o bin/coordinator ./cmd/coordinator
	go build -o bin/worker ./cmd/worker

dev.run: dev.build-local
	./bin/coordinator --v=5 &
	./bin/worker --coordinatorhost localhost:8000

dev.run-docker:
	docker build . -t cloud-game-local
	docker stop cloud-game-local || true
	docker rm cloud-game-local || true
	# Coordinator and worker should be run separately.
	docker run --privileged -v $(pwd)/games:/cloud-game/games -d --name cloud-game-local -p 8000:8000 -p 9000:9000 cloud-game-local bash -c "coordinator --v=5 & worker --coordinatorhost localhost:8000"

# RELEASE
# Builds the app for new release.
#
# Folder structure:
# release
#   - coordinator/
#     - web/
#     - coordinator
#   - worker/
#     - assets/
#       - emulator/libretro/cores/ (filtered by extension)
#       - games/
#     - worker
#
# params:
# - RELEASE_DIR: the name of the output folder (default: _release).
# - DLIB_TOOL: the name of a dynamic lib copy tool (with params) (e.g., ldd -x -y; defalut: ldd).
# - DLIB_SEARCH_PATTERN: a grep filter of the output of the DLIB_TOOL (e.g., mylib.so; default: .*so).
#   Be aware that this search pattern will return only matched regular expression part and not the whole line.
#   de. -> abc def ghj -> def
#   Makefile special symbols should be escaped with \.
# - DLIB_ALTER: a special flag to use altered dynamic copy lib tool for macOS only.
# - CORE_EXT: a file extension of the cores to copy into the release.
#
# example:
#   make release DLIB_TOOL="ldd -x" DLIB_SEARCH_PATTERN=/usr/lib.*\\\\s LIB_EXT=so
#
RELEASE_DIR ?= release
DLIB_TOOL ?= ldd
DLIB_SEARCH_PATTERN ?= .*so
DLIB_ALTER ?= false
CORE_EXT ?= *
COORDINATOR_DIR = ./$(RELEASE_DIR)/coordinator
WORKER_DIR = ./$(RELEASE_DIR)/worker
CORES_DIR = assets/emulator/libretro/cores
GAMES_DIR = assets/games
.PHONY: release
.SILENT: release
release: clean build
	rm -rf ./$(RELEASE_DIR) && mkdir ./$(RELEASE_DIR)
	mkdir $(COORDINATOR_DIR) && mkdir $(WORKER_DIR)
	cp ./bin/coordinator $(COORDINATOR_DIR) && cp ./bin/worker $(WORKER_DIR)
	chmod +x $(COORDINATOR_DIR)/coordinator $(WORKER_DIR)/worker
    ifeq ($(DLIB_ALTER),false)
		for bin in $$($(DLIB_TOOL) $(WORKER_DIR)/worker | grep -o $(DLIB_SEARCH_PATTERN)); \
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
    ifneq (,$(wildcard ./$(CORES_DIR)/*.$(CORE_EXT)))
		cp -R ./$(CORES_DIR)/*.$(CORE_EXT) $(WORKER_DIR)/$(CORES_DIR)
    endif
