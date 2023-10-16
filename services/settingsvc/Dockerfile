##### sdk image #####
FROM golang:1.21.1 AS sdk

WORKDIR /app

# Copy everything, respecting .dockerignore.
COPY . .

# Change to the directory of the service.
WORKDIR /app/services/settingsvc
RUN go mod download

# Build the service. The output binary is named "settingsvc".
RUN CGO_ENABLED=0 GOOS=linux go build -o settingsvc

###### release image #####
FROM alpine:latest

COPY --from=sdk /app/services/settingsvc/settingsvc /usr/local/bin/

ENTRYPOINT ["settingsvc"]
CMD ["-v=9", "-logtostderr"]