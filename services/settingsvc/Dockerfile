##### sdk image #####
FROM golang:1.21.1 AS sdk

WORKDIR /app

# Copy everything, respecting .dockerignore.
COPY . .

# Change to the directory of the service.
WORKDIR /app/services/settingsvc
RUN go mod download

# Build the service. The output binary is named "authnsvc".
RUN CGO_ENABLED=0 GOOS=linux go build -o settingsvc


#RUN ls -lart && go build -o /go/bin/gargantua main.go
###### release image #####
FROM alpine:latest

COPY --from=sdk /app/services/settingsvc/settingsvc /usr/local/bin/

ENTRYPOINT ["settingsvc"]
CMD ["-v=9", "-logtostderr"]