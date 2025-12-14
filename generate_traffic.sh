#!/usr/bin/env bash

URLS=(
  "http://localhost:9000/foo"
  "http://localhost:9000/maj"
)

while true; do
  for url in "${URLS[@]}"; do
    curl -s -o /dev/null "$url"
  done
  sleep 1
done