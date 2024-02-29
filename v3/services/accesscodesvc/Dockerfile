##### sdk image #####
FROM golang:1.21.1 AS sdk

# Change to the directory of the service.
WORKDIR /app/v3/services/accesscodesvc

COPY /v3/services/accesscodesvc/go.mod .
COPY /v3/go.mod /app/v3

RUN go mod download

# Copy everything, respecting .dockerignore.
COPY . /app

# Build the service. The output binary is named "app".
RUN CGO_ENABLED=0 GOOS=linux go build -o /tmp/app

###### release image #####
FROM alpine:latest

COPY --from=sdk /tmp/app /usr/local/bin/

ENTRYPOINT ["app"]
CMD ["-v=9", "-logtostderr"]