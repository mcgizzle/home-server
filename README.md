# home-server

The repo is split into `apps` and `infra` directories. The `apps` directory contains all the applications that run on the server. The `infra` directory contains all the infrastructure code that provisions the server.

Within `apps`, there are sets of services for two hosts:
1. `primary` - The primary host that runs most services (e.g. plex, *arr, etc.)
2. `network` - The network host that runs services that are critical for the server to function. (e.g. DNS, P2P VPN)

## Deploy
```shell
./deploy.sh <primary|network> [--pull-only] [--restart]
```