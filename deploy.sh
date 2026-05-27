#!/bin/bash

# Stop the script if any command fails (optional, good for safety)
# set -e

echo "========================================"
echo "Starting deployment on Server 1: cloudretro.io"
echo "========================================"

# SSH into cloudretro.io
# 1. Change directory to /cloud-game
# 2. Run docker compose commands using && to ensure order
ssh cloudretro.io "cd /cloud-game && docker compose pull && docker compose down && docker compose up -d"

echo ""
echo "========================================"
echo "Starting deployment on Server 2: 85.221.11.84"
echo "========================================"

# SSH into root@85.221.11.84
# No cd command needed (runs in home directory by default)
ssh root@85.221.11.84 "docker compose pull && docker compose down && docker compose up -d"

echo ""
echo "========================================"
echo "All deployments finished."
echo "========================================"