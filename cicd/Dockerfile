## Should only be used in the context of GitHub Actions pipeline
## If trying to build locally, use Dockerfile-local
FROM alpine:latest

COPY app /usr/local/bin/

ENTRYPOINT ["app"] 
CMD ["-v=9", "-logtostderr"] 
