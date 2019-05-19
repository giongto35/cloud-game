#!/bin/bash
# Run coordinator first
go run cmd/main.go -overlordhost overlord &
# Wait till overlord finish initialized
# Run a worker connecting to overlord
sleep 3s
go run cmd/main.go -overlordhost ws://localhost:8000/wso
