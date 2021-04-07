#!/usr/bin/env bash

echo This script should install application dependencies for Debian-based systems
if [ $(id -u) -ne 0 ]
then
  echo "error: run with sudo or root"
  exit 1
fi

apt-get -qq update
apt-get -qq install --no-install-recommends -y \
    libvpx6 \
    libx264-160 \
    libopus0 \
    libgl1-mesa-glx \
    xvfb
