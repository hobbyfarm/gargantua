FROM golang:1.13-alpine

# install deps
RUN apk add --no-cache \
        curl \
        git \
        jq \
        nodejs \
        npm \
    && npm install -g chokidar-cli@2.1.0

# chokidar needs the SHELL env var
ENV SHELL=/bin/sh

# install fixuid
RUN curl -SsL https://github.com/boxboat/fixuid/releases/download/v0.4/fixuid-0.4-linux-amd64.tar.gz | tar -C /usr/local/bin -xzf - \
    && chown root:root /usr/local/bin/fixuid \
    && chmod 4755 /usr/local/bin/fixuid

# add alpine user create/chown directories
RUN addgroup -g 1000 alpine \
    && adduser -u 1000 -G alpine -h /home/alpine -s /bin/sh -D alpine \
    && chown -R alpine:alpine /go \
    && mkdir -p /var/run/secrets/kubernetes.io/serviceaccount \
    && chown -R alpine:alpine /var/run/secrets/kubernetes.io/serviceaccount

USER alpine:alpine

# copy staged files
COPY stage/ /

WORKDIR /app
ENTRYPOINT ["fixuid"]
CMD ["run.sh"]
