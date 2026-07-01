# Troubleshooting

Common errors and how to resolve them.

---

## Installation Errors

### `bbe-networking`: required IP address values missing

**Error message:**

```
INSTALLATION FAILED: values don't meet the specifications of the schema(s) in the following chart(s):
bbe-networking:
- bbe.metallb.blocky.ipAddressPool: Invalid type. Expected: string, given: null
- ingress-nginx.controller.service.loadBalancerIP: String length must be greater than or equal to 1
```

**Cause:**

`bbe-networking` requires two static IP addresses to be configured before installation. These IPs are reserved on your local network for:

| Value | Purpose |
|---|---|
| `bbe.metallb.blocky.ipAddressPool` | Dedicated IP for the Blocky DNS service (LoadBalancer) |
| `ingress-nginx.controller.service.loadBalancerIP` | Dedicated IP for the ingress-nginx LoadBalancer |

These must be IP addresses that are **outside your router's DHCP range** so they are never automatically assigned to another device.

**When using `bbe install` (normal flow):**

The CLI detects these requirements automatically by reading `bbe-networking`'s `values.schema.json`. You are prompted for each value interactively, and the answers are saved to `~/.bbe/bbe-networking-values.yaml`. This file is reused on every subsequent `bbe upgrade`, so you are only prompted once.

If you see this error despite being prompted, it means you provided an empty value. Re-run `bbe install`, delete `~/.bbe/bbe-networking-values.yaml` first if it was written with empty values:

```bash
rm ~/.bbe/bbe-networking-values.yaml
bbe install
```

**When installing manually with Helm:**

1. Choose two free static IPs on your local network. For example:
   - `192.168.1.50` for Blocky DNS
   - `192.168.1.51` for ingress-nginx

2. Create a values override file (e.g. `~/.bbe/bbe-networking-values.yaml`):

   ```yaml
   bbe:
     metallb:
       blocky:
         ipAddressPool: "192.168.1.50-192.168.1.50"

   ingress-nginx:
     controller:
       service:
         loadBalancerIP: "192.168.1.51"
   ```

3. Install using the values file:

   ```bash
   helm install bbe-networking bbe/bbe-networking -f ~/.bbe/bbe-networking-values.yaml
   ```

> [!TIP]
> The `ipAddressPool` value supports a single IP (`192.168.1.50-192.168.1.50`) or a range (`192.168.1.50-192.168.1.59`). For a single Blocky instance, a single IP is sufficient.

> [!NOTE]
> Once set, point your router's DNS server (or individual device DNS) at the Blocky IP to enable network-wide ad-blocking.

---

## Setup Errors

### Node not discovered on the network

**Symptom:** `bbe setup` scans the subnet but does not find a Talos node in maintenance mode.

**Possible causes and fixes:**

- **Node is not booted from the Talos image** — confirm the device booted from the flashed drive, not from an existing OS on disk. Check the boot order in the BIOS.
- **Node is on a different subnet** — `bbe setup` scans the `/24` subnet of your machine's detected gateway. If the node booted onto a different VLAN or subnet, it will not be found automatically. Connect the node to the same network segment.
- **`nmap` lacks permissions** — on some systems, `nmap -sn` requires elevated permissions. Try running `bbe setup` with `sudo` or grant `nmap` the required capabilities.
- **Firewall blocking discovery** — ensure no firewall on the host machine is blocking ICMP or TCP traffic to the local subnet.

---

### `talosctl` config errors after re-provisioning a node

**Symptom:** `talosctl` commands fail after wiping and re-provisioning a node that previously existed.

**Fix:** Remove the stale entry from `~/.talos/config` and re-run `bbe setup`. Alternatively, delete `~/.bbe/talosconfig` and regenerate via setup.

---

## AWS S3 Errors

### Config sync fails with credential errors

**Symptom:**

```
ERROR  Failed to sync configs with AWS
```

**Fix:** Ensure AWS credentials are configured on the machine. BBE-Quest uses the standard AWS credential chain (`~/.aws/credentials`, environment variables, or instance profile). Run `aws sts get-caller-identity` to verify credentials are valid before re-running.
