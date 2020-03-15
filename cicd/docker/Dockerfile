##### sdk image #####
FROM golang:1.13-alpine AS sdk

WORKDIR /go/src/github.com/hobbyfarm/gargantua
COPY . .

ENV GOOS=linux 
ENV CGO_ENABLED=0

RUN go get -d -v ./...

RUN go install -v ./...


###### release image #####
FROM alpine:latest

COPY --from=sdk /go/bin/gargantua /usr/local/bin/

ENTRYPOINT ["gargantua"] 
CMD ["-v=9", "-logtostderr"] 
