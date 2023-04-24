#!/bin/sh

app="$1"

echo Making application runtime directories
mkdir -p ./assets/cache
mkdir -p ./assets/games
mkdir -p ./.cr
if [ "$app" = "worker" ]; then
  mkdir -p ./assets/cores
  mkdir -p ./libretro
fi


