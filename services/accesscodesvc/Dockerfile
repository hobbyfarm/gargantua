##### sdk image #####
FROM golang:1.21.1 AS sdk

WORKDIR /app

# Copy everything, respecting .dockerignore.
COPY . .

# Change to the directory of the service.
WORKDIR /app/services/accesscodesvc
RUN go mod download

# Build the service. The output binary is named "accesscodesvc".
RUN CGO_ENABLED=0 GOOS=linux go build -o accesscodesvc

###### release image #####
FROM alpine:latest

COPY --from=sdk /app/services/accesscodesvc/accesscodesvc /usr/local/bin/

ENTRYPOINT ["accesscodesvc"]
CMD ["-v=9", "-logtostderr"]