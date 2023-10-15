PROJECT = cloud-game
REPO_ROOT = github.com/giongto35
ROOT = ${REPO_ROOT}/${PROJECT}

CGO_CFLAGS='-g -O3 -funroll-loops'
CGO_LDFLAGS='-g -O3'
GO_TAGS=static

.PHONY: clean test

fmt:
	@goimports -w cmd pkg tests
	@gofmt -s -w cmd pkg tests

compile: fmt
	@go install ./cmd/...

clean:
	@rm -rf bin
	@rm -rf build
	@go clean ./cmd/*


build.coordinator:
	mkdir -p bin/
	go build -ldflags "-w -s -X 'main.Version=$(GIT_VERSION)'" -o bin/ ./cmd/coordinator

build.worker:
	mkdir -p bin/
	CGO_CFLAGS=${CGO_CFLAGS} CGO_LDFLAGS=${CGO_LDFLAGS} \
		go build -pgo=auto -buildmode=exe $(if $(GO_TAGS),-tags $(GO_TAGS),) \
		-ldflags "-w -s -X 'main.Version=$(GIT_VERSION)'" $(EXT_WFLAGS) \
		-o bin/ ./cmd/worker

build: build.coordinator build.worker

test:
	go test -v ./pkg/...

verify-cores:
	go test -run TestAll ./pkg/worker/room -v -renderFrames $(GL_CTX) -outputPath "./_rendered"

dev.build: compile build

dev.build-local:
	mkdir -p bin/
	go build -o bin/ ./cmd/coordinator
	CGO_CFLAGS=${CGO_CFLAGS} CGO_LDFLAGS=${CGO_LDFLAGS} go build -pgo=auto -o bin/ ./cmd/worker

dev.run: dev.build-local
ifeq ($(OS),Windows_NT)
	./bin/coordinator.exe &	./bin/worker.exe
else
	./bin/coordinator &	./bin/worker
endif

dev.run.debug:
	go build -race -o bin/ ./cmd/coordinator
	CGO_CFLAGS=${CGO_CFLAGS} CGO_LDFLAGS=${CGO_LDFLAGS} \
		go build -race -gcflags=all=-d=checkptr -o bin/ ./cmd/worker
	./bin/coordinator &	./bin/worker

dev.run-docker:
	docker rm cloud-game-local -f || true
	docker compose up --build

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
# - DLIB_TOOL: the name of a dynamic lib copy tool (with params) (e.g., ldd -x -y; default: ldd).
# - DLIB_SEARCH_PATTERN: a grep filter of the output of the DLIB_TOOL (e.g., my_lib.so; default: .*so).
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
release: GIT_VERSION := $(shell ./scripts/version.sh)
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
	# add version tag into index.html
	./scripts/version.sh $(COORDINATOR_DIR)/web/index.html
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
