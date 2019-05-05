#!/bin/bash
docker build . -t cloud-game-local
docker stop cloud-game-local
docker rm cloud-game-local
docker run --privileged -d --name cloud-game-local -p 8000:8000 cloud-game-local cmd -debug
