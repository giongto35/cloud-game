dep:
	go mod download
	go mod vendor
	go mod tidy
# NOTE: there is problem with go mod vendor when it delete github.com/gen2brain/x264-go/x264c causing unable to build. https://github.com/golang/go/issues/26366

build:
	go build -o build/cloudretro ./cmd

run: build
	# Run coordinator first
	./build/cloudretro -overlordhost overlord &
	# Wait till overlord finish initialized
	# Run a worker connecting to overload
	./build/cloudretro -overlordhost ws://localhost:8000/wso

run-docker:
	docker build . -t cloud-game-local
	docker stop cloud-game-local || true
	docker rm cloud-game-local || true
	# Overlord and worker should be run separately. Local is for demo purpose
	docker run --privileged -v $(PWD)/games:/cloud-game/games -d --name cloud-game-local -p 8000:8000 -p 9000:9000 cloud-game-local bash -c "cmd -overlordhost ws://localhost:8000/wso & cmd -overlordhost overlord"

#run with vendor will be faster
build-vendor:
	go build -o build/cloudretro -mod=vendor ./cmd

run-fast: build-vendor
	./build/cloudretro -overlordhost overlord &
	./build/cloudretro -overlordhost ws://localhost:8000/wso
