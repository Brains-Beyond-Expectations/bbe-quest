# `bbe config`

**Alias:** `bbe c`

Initialises or synchronises the BBE-Quest configuration file (`~/.bbe/bbe.yaml`).

---

## Usage

```bash
bbe config
```

---

## Flow

1. Checks whether `~/.bbe/bbe.yaml` already exists.
2. If it does not exist, runs an interactive wizard to create it:
   - **Storage type**: `local` or `aws`
   - If `aws`: prompts for the S3 bucket name
3. If `aws` storage is configured and the S3 bucket already contains config files, offers to sync them down to `~/.bbe/`.
4. Writes or updates `~/.bbe/bbe.yaml`.

---

## Storage Backends

### Local (default)

Config files (`controlplane.yaml`, `worker.yaml`, `talosconfig`) are stored only on the local machine in `~/.bbe/`. This is the simplest option for single-machine setups.

### AWS S3

Config files are also pushed to an S3 bucket after each change. This allows you to:
- Access your cluster config from multiple machines
- Recover from a lost `~/.bbe/` directory

The S3 bucket is automatically created (if it does not exist) with:
- **Server-side encryption** (SSE-S3) enabled
- **Versioning** enabled

---

## Config File Schema

See [Configuration](../configuration.md) for the full `bbe.yaml` schema.
