##### BUILD STAGE #####
FROM golang:1.21.13-alpine3.20 AS build

WORKDIR /app
# copy over dependency files and download dependencies
COPY go.mod .

RUN go mod download

# copy over source files
COPY . .

## build the service and output the binary to /tmp/app
RUN CGO_ENABLED=0 GOOS=linux go build -o /tmp/app

##### RUNTIME STAGE #####
FROM alpine:3.21.2

# create group and user app
RUN addgroup -S app && adduser -S app -G app

# copy over app binary from build stage
COPY --from=build /tmp/app /home/app/app
# make the binary executable
RUN chmod +x /home/app/app && chown -R app:app /home/app/app

# switch to user app
USER app
WORKDIR /home/app

ENTRYPOINT ["/home/app/app"]
CMD ["-v=9", "-logtostderr"]