## Should only be used in the context of GitHub Actions pipeline
## If trying to build locally, use Dockerfile-local
FROM alpine:3.21.2

# create group and user app
RUN addgroup -S app && adduser -S app -G app

# copy over app binary
COPY app /home/app/app
# make the binary executable
RUN chmod +x /home/app/app && chown -R app:app /home/app/app

# switch to user app
USER app
WORKDIR /home/app

ENTRYPOINT ["/home/app/app"]
CMD ["-v=9", "-logtostderr"]
