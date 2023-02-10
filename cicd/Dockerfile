## Should only be used in the context of GitHub Actions pipeline
## If trying to build locally, use Dockerfile-local
FROM alpine:latest

COPY gargantua /usr/local/bin/

ENTRYPOINT ["gargantua"] 
CMD ["-v=9", "-logtostderr"] 
