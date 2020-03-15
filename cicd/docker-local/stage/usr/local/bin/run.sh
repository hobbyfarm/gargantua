#!/bin/sh

set -e
cd /app

trap : TERM INT

go-run.sh &
pid=$!

(
    chokidar \
        -c "kill -USR2 $pid" \
        "go.mod" \
        "**/*.go" \
        "/var/run/secrets/kubernetes.io/serviceaccount/*"
) &

wait
