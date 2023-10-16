##### sdk image #####
FROM golang:1.20.2 AS sdk

WORKDIR /go/src/github.com/hobbyfarm/gargantua/services/conversionsvc
COPY . .

ENV GOOS=linux
ENV CGO_ENABLED=0

RUN go get -d -v ./...
RUN go install -v ./...

#RUN ls -lart && go build -o /go/bin/gargantua main.go
###### release image #####
FROM alpine:latest

COPY --from=sdk /go/bin/conversionsvc /usr/local/bin/

ENTRYPOINT ["conversionsvc"]
CMD ["-v=9", "-logtostderr"]