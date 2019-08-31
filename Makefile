dep:
	go mod download
	go mod vendor
	go mod tidy

build: dep
	go build -o build/cloudretro ./cmd

run: build
	# Run coordinator first
	./build/cloudretro -overlordhost overlord &
	# Wait till overlord finish initialized
	# Run a worker connecting to overload
	./build/cloudretro -overlordhost ws://localhost:8000/wso

run-docker:
	docker build . -t cloud-game-local
	docker stop cloud-game-local
	docker rm cloud-game-local
	# Overlord and worker should be run separately. Local is for demo purpose
	docker run --privileged -v $PWD/games:/cloud-game/games -d --name cloud-game-local -p 8000:8000 -p 9000:9000 cloud-game-local bash -c "cmd -overlordhost ws://localhost:8000/wso & cmd -overlordhost overlord"
