#!/bin/sh
GOOS=linux CGO_ENABLED=0 go build -o bin/gargantua
docker build -t oats87/gargantua:dev .
