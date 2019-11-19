FROM golang:latest as builder

WORKDIR /go/gargantua/
COPY . .
ENV GOOS=linux 
ENV CGO_ENABLED=0 
RUN go build -o bin/gargantua .
RUN touch /tmp

FROM scratch  
WORKDIR /app/
COPY --from=builder /go/gargantua/bin/gargantua .
COPY --from=builder /tmp/ /tmp/

ENTRYPOINT ["/app/gargantua"] 
CMD ["-v=9", "-logtostderr"]