#!/usr/bin/env bash

echo This script should install application dependencies for Debian-based systems
if [ $(id -u) -ne 0 ]
then
  echo "error: run with sudo or root"
  exit 1
fi

apt-get update
apt-get install --no-install-recommends -y \
    libvpx6 \
    libx264-160 \
    libopus0 \
    libopusfile0 \
    libsdl2-2.0-0 \
    libgl1-mesa-glx \
    xvfb
