FROM golang:1.13 as builder 
WORKDIR /go/src/github.com/hobbyfarm/gargantua

ENV GOOS=linux 
ENV CGO_ENABLED=0

COPY . .
RUN go install -v -mod=vendor


FROM alpine:3.11
	
COPY --from=builder /go/bin/gargantua /usr/local/bin/

ENTRYPOINT ["gargantua"] 
CMD ["-v=9", "-logtostderr"] 
