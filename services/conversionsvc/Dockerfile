##### sdk image #####
FROM golang:1.21.1 AS sdk

WORKDIR /app

# Copy everything, respecting .dockerignore.
COPY . .

# Change to the directory of the service.
WORKDIR /app/services/conversionsvc
RUN go mod download

# Build the service. The output binary is named "conversionsvc".
RUN CGO_ENABLED=0 GOOS=linux go build -o conversionsvc

###### release image #####
FROM alpine:latest

COPY --from=sdk /app/services/conversionsvc/conversionsvc /usr/local/bin/

ENTRYPOINT ["conversionsvc"]
CMD ["-v=9", "-logtostderr"]