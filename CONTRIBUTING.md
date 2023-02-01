# Contributing

## Code logic

It all starts from `main.go` file.

## Local Development

* Go 1.19 must be installed

```bash
go version
```

* Build the application

```bash
go build
```

* Start local Kubernetes cluster

```bash
k3d cluster create hobbyfarm --api-port 6550 -p "8081:80@loadbalancer" -p "8082:443@loadbalancer" --agents 1
```

* Create CA and TLS files (see [How To Create CA and Generate SSL/TLS Certificates & Keys](https://scriptcrunch.com/create-ca-tls-ssl-certificates-keys/))

```bash
cd .ssl/
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes \
      -key ca.key -subj "/CN=hobbyfarm/C=US/L=CALIFORNIA" \
      -days 1825 -out ca.crt
openssl genrsa -out server.key 2048
cat > csr.conf <<EOF
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
C = US
ST = California
L = San Fransisco
O = HobbyFarm
OU = HobbyFarm Dev
CN = hobbyfarm.github.io

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = hobbyfarm
DNS.2 = hobbyfarm.dev.local
EOF
openssl req -new -key server.key -out server.csr -config csr.conf
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out server.crt -days 10000 \
  -extfile csr.conf
cd ..
```

* Run the application

```bash
go run . --kubeconfig=%userprofile%\.kube\config --webhook-tls-cert=.ssl/server.crt --webhook-tls-key=.ssl/server.key -webhook-tls-ca=.ssl/ca.crt
```

## Local Development via docker-compose

First, start the docker-compose stack in [hobbyfarm/hobbyfarm](https://github.com/hobbyfarm/hobbyfarm) to provide a local [kind](https://github.com/kubernetes-sigs/kind) cluster for CRDs.

Next, run:

```bash
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

The script `./compose-up.sh` does the following:

- connects to the external docker network `hobbyfarm-dev`
- mounts the external volume for kube service account credentials called `hobbyfarm-kube-sa`
- calls `docker-compose up`
  - creates or starts the `hf-garg` container, which runs a watch loop on golang files, re-builds on change, and listens on [localhost:16210](http://localhost:16210)

To modify docker-compose variables for your local environment, copy `.env.example` to `.env` and update variables as needed.
