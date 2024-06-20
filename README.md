# Gargantua

[![CI](https://github.com/hobbyfarm/gargantua/actions/workflows/ci.yaml/badge.svg?branch=master)](https://github.com/hobbyfarm/gargantua/actions/workflows/ci.yaml)
[![PKG](https://github.com/hobbyfarm/gargantua/actions/workflows/pkg.yaml/badge.svg?branch=master)](https://github.com/hobbyfarm/gargantua/actions/workflows/pkg.yaml)
[![Docker Image Version (latest semver)](https://img.shields.io/docker/v/hobbyfarm/gargantua?label=Docker)](https://hub.docker.com/r/hobbyfarm/gargantua)

Gargantua is the megalith backend component of HobbyFarm, which is currently being built as a monolith, but will be transitioned into a proper microservices architecture one day™

## Design

Gargantua is based on, and provisions, the following Kubernetes controllers to manage various aspects of HobbyFarm:

- Terraform Provisioner Controller
- VMClaim Controller
- VMSet Controller
- Environment Controller
- Session Controller

Their relationship is illustrated with the following ERD:

![workflow](./artifacts/diagrams/Workflow.png)

## Contributions

If you're interested in working with the source code, see [CONTRIBUTING.md](CONTRIBUTING.md)
