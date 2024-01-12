##### sdk image #####
FROM golang:1.21.1 AS sdk

WORKDIR /app

# Copy everything, respecting .dockerignore.
COPY . .

# Change to the directory of the service.
WORKDIR /app/services/authnsvc
RUN go mod download

# Build the service. The output binary is named "app".
RUN CGO_ENABLED=0 GOOS=linux go build -o /tmp/app

###### release image #####
FROM alpine:latest

COPY --from=sdk /tmp/app /usr/local/bin/

ENTRYPOINT ["app"]
CMD ["-v=9", "-logtostderr"]