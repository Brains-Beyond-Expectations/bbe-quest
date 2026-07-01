# Supported Hardware

BBE-Quest is designed to run on commodity home-lab hardware. The following node types are supported:

---

## Intel NUC (x86_64)

| Attribute | Value |
|---|---|
| Architecture | `amd64` / `x86_64` |
| Image type | ISO (`.iso`) |
| Source | `factory.talos.dev` |
| Flash target | USB drive |

The ISO image is flashed to a USB drive with balenaEtcher and booted from the NUC's boot menu.

> [!NOTE]
> Talos does not support Secure Boot on x86. Disable Secure Boot in the BIOS before booting the Talos image.

---

## Raspberry Pi 4 (ARM64)

| Attribute | Value |
|---|---|
| Architecture | `arm64` |
| Image type | Raw disk image (`.raw.xz`) |
|Source | `factory.talos.dev` |
| Flash target | microSD card or USB drive |

The raw image is written directly to a microSD card or USB drive. Raspberry Pi 4 boots from the first available storage device.

---

## Mixed Clusters

You can mix Intel NUC and Raspberry Pi 4 nodes in the same cluster. Run `bbe setup` separately for each node and select the appropriate hardware type each time. The control plane can be on either hardware type.

---

## Future Hardware

Support for additional hardware targets may be added in future releases by extending the `NodeType` model. See [Architecture](architecture.md) for details.
