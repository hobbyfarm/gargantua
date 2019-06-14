FROM ubuntu:16.04

COPY gargantua /usr/local/bin

ENTRYPOINT /usr/local/bin/gargantua -v=9 -alsologtostderr
