# `bbe install`

**Alias:** `bbe i`

Interactive command to install or uninstall Helm package bundles on the cluster. Presents a multi-select menu of all packages available for the running CLI version.

---

## Usage

```bash
bbe install
# or
bbe i
```

---

## Flow

1. Loads `~/.bbe/bbe.yaml` to get the configured Kubernetes context.
2. Fetches [`library.yaml`](https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-charts/main/library.yaml) from the bbe-charts repository.
3. Selects the most recent library entry compatible with the running CLI version.
4. Displays a multi-select TUI listing all available packages. Packages already installed are pre-selected.
5. For each package that was **selected and not previously installed**:
   - If the chart has a `values.schema.json` with required fields, the CLI prompts for each required value and saves the answers to `~/.bbe/<package>-values.yaml`.
   - Runs `helm repo add` + `helm install`, passing the values file if one was created.
6. For each package that was **deselected and previously installed** → runs `helm uninstall`.
7. Updates `bbe.yaml` with the new list of installed packages.

## Package Configuration

Some packages require configuration before they can be installed. The CLI detects required fields from the chart's `values.schema.json` and prompts interactively:

```
Package `bbe-networking` requires configuration. Please provide the following values:
bbe.metallb.blocky.ipAddressPool (The IP address pool to use for the Blocky service): 192.168.1.50-192.168.1.50
ingress-nginx.controller.service.loadBalancerIP (The IP address to use for the load balancer): 192.168.1.51
Configuration saved to /home/user/.bbe/bbe-networking-values.yaml
```

The values file is saved to `~/.bbe/<package>-values.yaml` and reused on every subsequent `bbe upgrade` — you are only prompted once. To reconfigure, delete the file and re-run `bbe install`:

```bash
rm ~/.bbe/bbe-networking-values.yaml
bbe install
```

---

## Available Packages

Packages are defined in [bbe-charts](https://github.com/Brains-Beyond-Expectations/bbe-charts):

| Package | Description |
|---|---|
| `bbe-media` | Full media server stack (Jellyfin, Sonarr, Radarr, Prowlarr, Bazarr, Jellyseerr) |
| `bbe-networking` | Network infrastructure (MetalLB, ingress-nginx, Blocky DNS) |

---

## Requirements

- A healthy Talos cluster provisioned with [`bbe setup`](setup.md)
- `helm` installed and available on `$PATH`
- The Kubernetes context configured in `~/.bbe/bbe.yaml`
