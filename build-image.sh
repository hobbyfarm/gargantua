#!/bin/sh
GOOS=linux CGO_ENABLED=0 go build
docker build -t oats87/gargantua:hfv2 .
