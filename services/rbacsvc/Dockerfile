##### sdk image #####
FROM golang:1.21.1 AS sdk

WORKDIR /app

# Copy everything, respecting .dockerignore.
COPY . .

# Change to the directory of the service.
WORKDIR /app/services/rbacsvc
RUN go mod download

# Build the service. The output binary is named "rbacsvc".
RUN CGO_ENABLED=0 GOOS=linux go build -o rbacsvc

###### release image #####
FROM alpine:latest

COPY --from=sdk /app/services/rbacsvc/rbacsvc /usr/local/bin/

ENTRYPOINT ["rbacsvc"]
CMD ["-v=9", "-logtostderr"]