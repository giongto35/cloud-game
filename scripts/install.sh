#!/usr/bin/env bash

echo This script should install application dependencies for Debian-based systems
if [ $(id -u) -ne 0 ]
then
  echo "error: run with sudo or root"
  exit 1
fi

apt-get -qq update
apt-get -qq install --no-install-recommends -y \
    ca-certificates \
    libvpx7 \
    libx264-164 \
    libopus0 \
    libgl1-mesa-dri \
    xvfb
apt-get clean
apt-get autoremove
