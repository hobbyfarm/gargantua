#!/bin/sh -e

cd $(dirname $0)

build_arg=""
script="$0"

up_usage() {
cat >&2 <<EOF
start HobbyFarm Gargantua

    usage: $script up <options>

options:
        --build  - build new container
    -h, --help   - print this message

EOF
}

up() {
    while test $# -gt 0
    do
        case "$1" in
            -h | --help)
                up_usage
                exit 0
                ;;
            --build)
                build_arg="--build"
                ;;
            *)
                up_usage
                exit 1
                ;;
        esac
        shift
    done

    # ensure hf-k3d is running
    k3d_state=$(docker container inspect hf-k3d -f '{{ .State.Status }}')
    if [ "$k3d_state" != "running" ]; then
        echo "container 'hf-k3d' must be running" >&2
        exit 1
    fi

    docker-compose up $build_arg
}

stop() {
    docker-compose stop
}

destroy() {
    docker-compose down -v
}

usage() {
cat >&2 <<-EOF
manage local HobbyFarm Gargantua development environment

        usage: $script <options> <command>
        
where <command> is one of:

    up          - create or start gargantua
    stop        - stop gargantua
    destroy     - destroy gargantua

options:
    -h, --help  - print this message

EOF
}

case "$1" in
    -h | --help)
        usage
        exit 0
        ;;
    up)
        shift
        up "$@"
        ;;
    stop)
        stop
        ;;
    destroy)
        destroy
        ;;
    *)
        usage
        exit 1
        ;;
esac
