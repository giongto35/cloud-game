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
#	go mod vendor
#	go mod tidy

# NOTE: there is problem with go mod vendor when it delete github.com/gen2brain/x264-go/x264c causing unable to build. https://github.com/golang/go/issues/26366
#build.cross: build
#	CGO_ENABLED=1 GOOS=darwin GOARC=amd64 go build --ldflags '-linkmode external -extldflags "-static"' -o bin/overlord-darwin ./cmd/overlord
#	CGO_ENABLED=1 GOOS=darwin GOARC=amd64 go build --ldflags '-linkmode external -extldflags "-static"' -o bin/overworker-darwin ./cmd/overworker
#	CC=arm-linux-musleabihf-gcc GOOS=linux GOARC=amd64 CGO_ENABLED=1 go build --ldflags '-linkmode external -extldflags "-static"' -o bin/overlord-linu ./cmd/overlord
#	CC=arm-linux-musleabihf-gcc GOOS=linux GOARC=amd64 CGO_ENABLED=1 go build --ldflags '-linkmode external -extldflags "-static"' -o bin/overworker-linux ./cmd/overworker

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
	@go clean

dev.tools:
	./hack/script/install_tools.sh

dev.build: compile
	go build -a -tags netgo -ldflags '-w' -o bin/overlord ./cmd/overlord
	go build -a -tags netgo -ldflags '-w' -o bin/overworker ./cmd/overworker

dev.build-local:
	go build -o bin/overlord ./cmd/overlord
	go build -o bin/overworker ./cmd/overworker

dev.run: dev.build-local
	./bin/overlord --v=5 &
	./bin/overworker --overlordhost ws://localhost:8000/wso



#dev.run: dev.build
#	# Run coordinator first
#	./build/klog -overlordhost overlord &
#	# Wait till overlord finish initialized
#	# Run a worker connecting to overload
#	./build/klog -overlordhost ws://localhost:8000/wso

#dev.run.docker:
#	docker build . -t cloud-game-local
#	docker stop cloud-game-local
#	docker rm cloud-game-local
#	# Overlord and worker should be run separately. Local is for demo purpose
#	docker run --privileged -v $PWD/games:/cloud-game/games -d --name cloud-game-local -p 8000:8000 -p 9000:9000 cloud-game-local bash -c "cmd -overlordhost ws://localhost:8000/wso & cmd -overlordhost overlord"

#build:
#	go build -o build/cloudretro ./cmd
#
#run: build
#	# Run coordinator first
#	./build/cloudretro -overlordhost overlord &
#	# Wait till overlord finish initialized
#	# Run a worker connecting to overload
#	./build/cloudretro -overlordhost ws://localhost:8000/wso

#run-docker:
#	docker build . -t cloud-game-local
#	docker stop cloud-game-local || true
#	docker rm cloud-game-local || true
#	# Overlord and worker should be run separately. Local is for demo purpose
#	docker run --privileged -v $(PWD)/games:/cloud-game/games -d --name cloud-game-local -p 8000:8000 -p 9000:9000 cloud-game-local bash -c "cmd -overlordhost ws://localhost:8000/wso & cmd -overlordhost overlord"
#

#run-fast: build-vendor
#	./build/cloudretro -overlordhost overlord &
#	./build/cloudretro -overlordhost ws://localhost:8000/wso
#
