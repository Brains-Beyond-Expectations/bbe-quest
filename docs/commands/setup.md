# `bbe setup`

Runs an interactive wizard to provision a new node and either bootstrap a fresh Talos cluster (first node) or join it as a worker (subsequent nodes).

---

## Usage

```bash
bbe setup
```

No flags are required. The wizard is fully interactive.

---

## Step-by-Step Flow

### 1. Config Initialisation
Loads `~/.bbe/bbe.yaml`. If configured with AWS S3, it optionally syncs existing cluster config files from the bucket first.

### 2. Dependency Check
Verifies that `talosctl`, `nmap`, `grep`, `bash`, and `awk` are available on `$PATH`. If any are missing, the command exits with instructions to install them.

### 3. Node Role Selection
Asks whether this is the **first node** (control plane) or an **additional node** (worker).

### 4. Hardware Selection
Asks whether the target hardware is an **Intel NUC** (x86, ISO image) or a **Raspberry Pi 4** (ARM, `.raw.xz` image).

### 5. Image Download
Downloads the appropriate pre-built Talos factory image from `factory.talos.dev`. Displays a progress indicator during download.

### 6. Flashing
Instructs the user to flash the downloaded image to a drive using **balenaEtcher**, then boot the target device. The wizard pauses and waits for confirmation.

### 7. Node Discovery
Automatically detects the local network gateway IP (WSL-aware via `ipconfig`). Runs `nmap -sn <gateway>/24` to discover hosts. Pings each discovered host with `talosctl machineconfig get` to identify Talos nodes in maintenance mode.

### 8. Node Configuration Prompts
Prompts for:
- **Static IP** — the IP to assign permanently to the node
- **Disk** — the disk to install Talos onto (fetched from the node via `talosctl`)
- **Network interface** — the interface to configure
- **Gateway IP** — confirmed or overridden
- **Hostname** — the node's hostname
- **Cluster name** (first node only)
- **Schedule workloads on control plane** (first node only) — enables running pods on the control plane

### 9. Config Generation and Application
- Runs `talosctl gen config` to create `controlplane.yaml`, `worker.yaml`, and `talosconfig` in `~/.bbe/`.
- Patches the generated YAML with the static IP, gateway, hostname, disk, and network interface.
- Runs `talosctl apply-config` to push the config to the node.

### 10. Bootstrap (First Node Only)
Runs `talosctl bootstrap` to initialise the etcd cluster.

### 11. Health Check
Polls `talosctl health` until the cluster is healthy (up to a 5-minute timeout).

### 12. Kubeconfig
Downloads the cluster kubeconfig via `talosctl kubeconfig ~/.kube/config` and saves the cluster name into `bbe.yaml`.

### 13. Config Persistence
If S3 storage is configured, uploads all config files to the S3 bucket.

---

## Config Files Created

All files are stored in `~/.bbe/`:

| File | Description |
|---|---|
| `controlplane.yaml` | Talos machine config for the control plane node |
| `worker.yaml` | Talos machine config for worker nodes |
| `talosconfig` | `talosctl` client configuration |
| `bbe.yaml` | Updated with the cluster name after bootstrap |
