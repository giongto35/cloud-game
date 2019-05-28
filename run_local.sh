#!/bin/bash
go build -o klog ./cmd
# Run coordinator first
./klog -overlordhost overlord &
# Wait till overlord finish initialized
# Run a worker connecting to overlord
./klog -overlordhost ws://localhost:8000/wso
# NOTE: Overlord and worker should be run separately. Local is for demo purpose
