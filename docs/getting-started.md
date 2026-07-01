# Getting Started

## Prerequisites

Before setting up your cluster, ensure you have the following installed on the machine you will run `bbe` from:

| Tool | Purpose | Install |
|---|---|---|
| [balenaEtcher](https://www.balena.io/etcher/) | Flash Talos images to USB drives or SD cards | [Download](https://www.balena.io/etcher/) |
| [talosctl](https://www.talos.dev/v1.8/learn-more/talosctl/) | Talos Linux CLI — used to configure and manage nodes | [Docs](https://www.talos.dev/v1.8/talos-guides/install/talosctl/) |
| [nmap](https://nmap.org/) | Network discovery — used to find nodes on the local subnet | [Download](https://nmap.org/download) |
| [helm](https://helm.sh/) | Kubernetes package manager — used to install charts | [Docs](https://helm.sh/docs/intro/install/) |

> [!NOTE]
> On x86 hardware (e.g., Intel NUC), Talos does not support Secure Boot. Disable Secure Boot in the BIOS before proceeding.

---

## Installing the BBE-Quest CLI

Run the install script with:

```bash
curl -fsSL https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-quest/main/install.sh | bash
```

This downloads the latest `bbe` binary and places it on your `$PATH`.

---

## First-Time Setup

### 1. Initialise Configuration

Run [`bbe config`](commands/config.md) to create the `bbe.yaml` configuration file:

```bash
bbe config
```

This creates `~/.bbe/bbe.yaml`. You can choose between local storage or AWS S3 for storing your cluster config files.

### 2. Add Your First Node (Control Plane)

Run [`bbe setup`](commands/setup.md) to provision your first node:

```bash
bbe setup
```

The wizard will guide you through:
1. Selecting hardware type (Intel NUC / Raspberry Pi 4)
2. Downloading the Talos factory image
3. Flashing the image to a drive with balenaEtcher
4. Auto-detecting the node on the network
5. Configuring the node (IP, hostname, disk)
6. Bootstrapping the Talos cluster and downloading `kubeconfig`

### 3. Install Packages

Run [`bbe install`](commands/install.md) to install the available package bundles:

```bash
bbe install
```

A multi-select menu lists the available packages. Select the ones you want and confirm to install them.

---

## Adding More Nodes

Re-run `bbe setup` for each subsequent worker node. The wizard will detect that a cluster already exists and configure the new node as a worker.

---

## Next Steps

- [Architecture](architecture.md) — Understand how BBE-Quest works internally
- [Commands Reference](commands/setup.md) — Detailed documentation for every command
- [Configuration](configuration.md) — `bbe.yaml` schema and storage options
- [Supported Hardware](hardware.md) — Intel NUC and Raspberry Pi 4 details
