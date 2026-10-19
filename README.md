# BBE-Quest: Big Brain Energy Quest

[![codecov](https://codecov.io/gh/Brains-Beyond-Expectations/bbe-quest/graph/badge.svg?token=Q7M8SJHDDW)](https://codecov.io/gh/Brains-Beyond-Expectations/bbe-quest)

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

- [balenaEtcher](https://www.balena.io/etcher/)
- [talosctl](https://www.talos.dev/v1.8/learn-more/talosctl/)
- [nmap](https://nmap.org/)

### Installing the BBE-Quest CLI

To install the BBE-Quest CLI, run the following command:

```bash
curl -fsSL https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-quest/main/install.sh | bash
```

### Setting up the cluster and its apps

Run `bbe setup` to install Talos on your nodes and create the cluster, then `bbe install` to add packages: storage,
networking, single sign-on, media, books and music.

[Getting started](https://github.com/Brains-Beyond-Expectations/bbe-charts/blob/main/docs/getting-started.md) walks
through the whole setup for new users: the NAS share, the addresses to keep free on your network, installing the
packages, making your devices find and trust the apps, adding your household, and each package's first steps.

| Command | What it does |
|---|---|
| `bbe setup` | Installs Talos on a node, and creates the cluster or adds the node to it |
| `bbe install` | Installs packages, and what they need first |
| `bbe upgrade` | Upgrades the installed packages |
| `bbe configure <package>` | Adds to and removes from a package's settings, such as the people in bbe-auth or the indexers in bbe-media |
| `bbe uninstall <package>` | Removes a package, after listing what it keeps |

Everything bbe installs, and its settings, is kept in `~/.bbe/bbe.yaml`, which sets up the same packages on a new
cluster.

## Local Development

Refer the requirements below and make sure you have Go version 1.23 or higher
installed. Change directory to the cli folder:

```bash
cd cli
```

To call a CLI command, run:

```bash
go run main.go <command>
```

To run the tests, run:

```bash
make test
```
