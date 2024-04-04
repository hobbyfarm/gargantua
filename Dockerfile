##### sdk image #####
FROM golang:1.21.1 AS sdk

WORKDIR /app
COPY go.mod .

# Change to the directory of the service.
RUN go mod download -x

COPY . .

# Build the service. The output binary is named "app".
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /tmp/app

#RUN ls -lart && go build -o /go/bin/gargantua main.go
###### release image #####
FROM alpine:latest

COPY --from=sdk /tmp/app /usr/local/bin/

ENTRYPOINT ["app"] 
CMD ["-v=9", "-logtostderr"] 
