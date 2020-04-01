#!/bin/sh

set -e
cd /app

garg_pid="0"
shell_pid="0"
restart="-1"
restart_request="0"

handle_usr2() {
    restart_request=$(( (restart + 1) % 1024 ))
}

wait_for_death() {
    for i in $(seq 1 "$2"); do
        if ! kill -0 "$1" >/dev/null 2>&1; then
            return 0
        fi
        sleep 0.1
    done
    return 1
}

kill_loop() {
    echo "restarting process" >&2
    kill "$1" >/dev/null 2>&1 || :
    if wait_for_death "$1" 50; then
        return 0
    fi

    echo "process did not terminate within 5s, trying again" >&2
    kill "$1" >/dev/null 2>&1 || :
    if wait_for_death "$1" 50; then
        return 0
    fi

    echo "process did not terminate within 10s, forcefully killing" >&2
    kill -kill "$1" >/dev/null 2>&1 || :
    if wait_for_death "$1" 50; then
        return 0
    fi

    echo "process did not respond to SIGKILL" >&2
    return 1
}

trap handle_usr2 USR2

while : ; do
    if [ "$restart" != "$restart_request" ]; then
        restart="$restart_request"
        
        # check if process is still running
        if ! kill -0 "$garg_pid" >/dev/null 2>&1; then
            garg_pid="0"
        fi
        if ! kill -0 "$shell_pid" >/dev/null 2>&1; then
            shell_pid="0"
        fi

        # kill process if still running
        if [ "$garg_pid" != "0" ]; then
            kill_loop "$garg_pid" &
        fi
        if [ "$shell_pid" != "0" ]; then
            kill_loop "$shell_pid" &
        fi
        wait

        # attempt to cmpile and start new process
        if go install -v ./...; then
            PORT=8080 gargantua -v=9 -logtostderr &
            garg_pid="$!"
            PORT=8081 gargantua -v=9 -logtostderr -shellserver -disablecontrollers &
            shell_pid="$!"
        else
            garg_pid="0"
            shell_pid="0"
        fi
    fi

    sleep 0.1
done
