#!/bin/bash
docker build . -t cloud-game-local
docker stop cloud-game-local
docker rm cloud-game-local-overlord -f
docker rm cloud-game-local-worker -f
docker run --privileged -d --name cloud-game-local-overlord -p 8000:8000 cloud-game-local cmd -overlordhost overlord
sleep 1s
docker run --privileged -d --name cloud-game-local-worker cloud-game-local cmd -overlordhost ws://localhost:8000/wso 
