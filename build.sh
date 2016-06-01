#!/bin/bash -x -e
docker build -t sip-ping-builder .
docker run --rm -v "$PWD":/usr/src/sip-ping -w /usr/src/sip-ping sip-ping-builder go build -v
