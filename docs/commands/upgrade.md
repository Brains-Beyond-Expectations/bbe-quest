# `bbe upgrade`

**Alias:** `bbe u`

Upgrades all installed Helm packages to the latest versions available for the running CLI version.

---

## Usage

```bash
bbe upgrade
# or with auto-confirm
bbe upgrade --yes
bbe upgrade -y
```

---

## Flags

| Flag | Short | Description |
|---|---|---|
| `--yes` | `-y` | Skip the confirmation prompt and upgrade immediately |

---

## Flow

1. Loads `~/.bbe/bbe.yaml` to get the list of installed packages and the Kubernetes context.
2. Fetches [`library.yaml`](https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-charts/main/library.yaml) from the bbe-charts repository.
3. Selects the most recent library entry compatible with the running CLI version.
4. For each installed package, resolves the latest chart version from the library entry.
5. Unless `--yes` is set, shows a confirmation prompt listing the packages to be upgraded.
6. Runs `helm repo update` + `helm upgrade` for each installed package.
7. Updates the version recorded for each package in `bbe.yaml`.

---

## When to Run

Run `bbe upgrade` periodically to pick up new chart versions released in bbe-charts. If the CLI itself is updated, upgrade again — a newer CLI version may unlock a newer library entry with updated chart versions.
