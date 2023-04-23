#!/usr/bin/env bash

deps="$1"

echo This script should install application dependencies for Debian-based systems
if [ $(id -u) -ne 0 ]
then
  echo "error: run with sudo or root"
  exit 1
fi

apt-get -qq update
if [ "$deps" = "x11-only" ]; then
  apt-get -qq install --no-install-recommends -y \
      libgl1-mesa-dri \
      xvfb
else
  apt-get -qq install --no-install-recommends -y \
      libvpx7 \
      libx264-164 \
      libopus0 \
      libgl1-mesa-dri \
      xvfb
fi
apt-get clean
apt-get autoremove
