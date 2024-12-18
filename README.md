# BBE-Quest: Big Brain Energy Quest

![BBE-Quest Banner](./assets/banner.webp)

BBE-Quest is a CLI tool that helps you easily set up a Kubernetes cluster using
Talos. It is designed to be a simple and easy-to-use tool that automates the
process of setting up a Kubernetes cluster on your hardware, including several
useful tools.

The goal of BBE-Quest is to be a set and forget way to setup your home lab
cluster.

## Getting Started

> [!NOTE]  
> Since Talos does not support secure boot on x86, you will need to disable
> secure boot in the BIOS settings of x86 devices.

### Requirements

- [Docker](https://docs.docker.com/get-docker/)
- [crane](https://github.com/google/go-containerregistry/blob/main/cmd/crane/README.md)
- [balenaEtcher](https://www.balena.io/etcher/)
- [talosctl](https://www.talos.dev/v1.8/learn-more/talosctl/)
- [nmap](https://nmap.org/)
- [yq](https://mikefarah.gitbook.io/yq/)

### Installing the BBE-Quest CLI

To install the BBE-Quest CLI, run the following command:

```bash
curl -fsSL https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-quest/main/install.sh | bash
```

## Local Development

Refer the requirements below and make sure you have Go version 1.23 or higher
installed. Change directory to the cli folder:

```bash
cd cli
```

Then to run a CLI command run:

```bash
go run main.go <command>
```

To run the tests, run:

```bash
go test ./...
```
