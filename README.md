# home-server

## Overview 🌎

This repo contains the code that provisions and manages my home server. The server is a Proxmox host that runs a number of LXC containers.

The LXC containers run Debian 12, provisioned with Ansible.

Apps are run from Docker containers within the LXC containers.

## Host 

Proxmox host is not setup nicely for development. Usually I just ssh or use the web interface.

## LXCs ⛴️

### Primary 📟

Runs most of my services including all media services.

### Network 🛜

DNS, point-to-point VPN, and reverse proxy.

## Deploy 🚀

```shell
./deploy.sh <primary|network> [--pull-only] [--restart] [--update]
```

`--update` will pull the latest Docker images.


## Development 🧑‍💻

All code is managed in git.

The servers filesystems are mounted using a 'Deployment' backed by SFTP. 

The `.idea` directory is purposely not ignored to make this repeatable. 

SSHing is easy with IntelliJ's terminal.


## TODO
- [ ]  Sort out users/groups
- [x]  Add ssh keys to 1pass 
- [ ]  Create LXC for side projects
- [ ]  Add proper metric dashboards
- [ ]  Improve docs