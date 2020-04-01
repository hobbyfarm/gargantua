# Contributing

## Local Development via docker-compose

First, start the docker-compose stack in [hobbyfarm/hobbyfarm](https://github.com/hobbyfarm/hobbyfarm) to provide a local  [kind](https://github.com/kubernetes-sigs/kind) cluster for CRDs.  Next, run:

```
# create or start stack
./compose.sh up

# -- or --
# start the stack, building changes to local dev container
# only needed if a file in ./cicd/docker-local has changed
./compose.sh up --build

# stop stack
./compose.sh stop

# destroy stack
./compose.sh destroy
```

`./compose-up.sh` does the following:

- connects to the external docker network `hobbyfarm-dev`
- mounts the external volume for kube service account credentials called `hobbyfarm-kube-sa`
- calls `docker-compose up`
    - creates or starts the `hf-garg` container, which runs a watch loop on golang files, re-builds on change, and listens on [localhost:16210](http://localhost:16210)

To modify docker-compose variables for your local environment, copy `.env.example` to `.env` and update variables as needed
