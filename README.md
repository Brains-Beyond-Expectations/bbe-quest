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
| `bbe upgrade` | Upgrades the installed packages, and applies the values you change in `~/.bbe/bbe.yaml` |
| `bbe configure <package>` | Adds to and removes from a package's settings, such as the people in bbe-auth or the indexers in bbe-media |
| `bbe uninstall <package>` | Removes a package, after listing what it keeps |

Everything bbe installs, and its settings, is kept in `~/.bbe/bbe.yaml`, which sets up the same packages on a new
cluster.

### What `bbe setup` locks down

When it creates a cluster, `bbe setup` asks which networks may reach the Talos and Kubernetes APIs, suggesting the
first node's `/24`. They have to include every node and the computers you manage the cluster from.

- **A firewall on every node** keeps the Talos API (50000, 50001), the Kubernetes API (6443), etcd, the kubelet
  (10250), Flannel's VXLAN (4789) and the metrics of the control plane and node-exporter to those networks, and to pods where they need them.
  Every other port, the apps' included, stays open.
- **NetworkPolicies are enforced**: Talos runs kube-network-policies alongside Flannel, so the policies packages bring
  limit which pods reach their apps.
- **The control plane's metrics** (etcd, the controller manager, the scheduler and kube-proxy) are served on the node's
  address, for a monitoring package to collect.

Nodes added later get the same settings, as they're kept in the configs in `~/.bbe`. A cluster created by an earlier
bbe keeps its settings; to lock it down too, apply the same documents with `talosctl patch machineconfig` to every
node, and the Flannel and metrics ones to the control plane.

> [!WARNING]
> Once the firewall is on, `talosctl`, `kubectl` and bbe only work from the networks you entered. A computer
> elsewhere can't reach the nodes, nor change their firewall.

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

### Testing charts that aren't released yet

bbe installs packages from the library of released charts in
[bbe-charts](https://github.com/Brains-Beyond-Expectations/bbe-charts). To install or upgrade with charts from a
bbe-charts pull request instead, point bbe at that pull request's preview library, from the `cli` folder:

```bash
go run . install --library https://github.com/Brains-Beyond-Expectations/bbe-charts/releases/download/preview-pr-<number>/library.yaml
go run . upgrade --library https://github.com/Brains-Beyond-Expectations/bbe-charts/releases/download/preview-pr-<number>/library.yaml
```

Or set it once for the whole shell, which every command then uses:

```bash
export BBE_LIBRARY=https://github.com/Brains-Beyond-Expectations/bbe-charts/releases/download/preview-pr-<number>/library.yaml
go run . install
```

- **The preview** is updated on every push to the pull request, whose comment shows its library and chart versions.
  A bbe built from source accepts any library, so use `go run .` rather than an installed `bbe`, which may be too old
  for the charts.
- **"chart ... version ... not found" right after a push** means helm's cached index of the preview is older than the
  library. Refresh it, then run the command again:

  ```bash
  helm repo update bbe-pr-<number>
  ```

- **Charts on disk, without pushing:** a library file can point at chart folders in a bbe-charts checkout, with paths
  relative to the file:

  ```yaml
  library:
    - min-bbe-cli: 0.0.0
      list-revision: 1
      charts:
        - name: bbe-music
          version: 0.0.0-local
          repositoryUrl: file://charts/bbe-music
          repositoryName: local
  ```

  ```bash
  go run . install --library ../../bbe-charts/local-library.yaml
  ```

Everything bbe installs is recorded in `~/.bbe/bbe.yaml`, preview versions included, so the next `go run . upgrade`
with the released library moves the packages back to released charts once they're out.
