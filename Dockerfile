##### BUILD STAGE #####
# use BUILDPLATFORM to pin to the native platform to prevent emulation from kicking in
FROM --platform=$BUILDPLATFORM golang:1.23.2-alpine3.20 AS build

# os from --platform linux/amd64
ARG TARGETOS
# architecture from --platform linux/amd64
ARG TARGETARCH

WORKDIR /app
# copy over dependency files and download dependencies
COPY go.mod .

RUN go mod download

# copy over source files
COPY . .

# build the service and output the binary to /tmp/app
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /tmp/app

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