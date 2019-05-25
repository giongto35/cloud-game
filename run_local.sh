#!/bin/bash
go build -o pogo ./cmd
# Run coordinator first
./pogo -overlordhost overlord &
# Wait till overlord finish initialized
# Run a worker connecting to overlord
./pogo -overlordhost ws://localhost:8000/wso
# NOTE: Overlord and worker should be run separately. Local is for demo purpose
