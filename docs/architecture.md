# Architecture

BBE-Quest is a Go CLI tool built with [Cobra](https://github.com/spf13/cobra). It follows a clean layered architecture where Cobra commands act as thin orchestrators and all business logic lives in service packages behind interfaces.

---

## High-Level Overview

```
User
 │
 ▼
Cobra CLI (cmd/)
 ├── setup    → DependencyService → ImageService → IpFinderService → TalosService → ConfigService
 ├── install  → PackageService (fetches library.yaml from bbe-charts) → HelmService → ConfigService
 ├── upgrade  → PackageService → HelmService → ConfigService
 ├── config   → ConfigService ↔ S3Service
 └── version  → constants.Version
```

Config is stored in `~/.bbe/` (locally) or in an AWS S3 bucket (remote mode).

---

## Project Structure

```
cli/
├── main.go             ← Entry point
├── cmd/                ← Cobra command definitions (thin orchestrators)
├── constants/          ← Shared constants, error messages, URLs
├── interfaces/         ← Go interfaces — each service is defined here
├── models/             ← Data structure definitions (mapped to YAML)
├── services/           ← Business logic (one package per service)
└── mocks/              ← Mock implementations for unit tests
```

---

## Services

| Service | Package | Responsibility |
|---|---|---|
| **ConfigService** | `config_service` | Read/write `~/.bbe/bbe.yaml`; push/pull config files to/from AWS S3 |
| **TalosService** | `talos_service` | Wraps `talosctl`: generate configs, apply-config, bootstrap, health check, get disks, patch YAML, download kubeconfig |
| **ImageService** | `image_service` | Downloads Talos factory images from `factory.talos.dev` (ISO for Intel NUC, `.raw.xz` for Raspberry Pi) |
| **IpFinderService** | `ipfinder_service` | Detects gateway IP, runs `nmap -sn /24`, identifies Talos nodes in maintenance mode |
| **PackageService** | `package_service` | Fetches `library.yaml` from bbe-charts; resolves the correct chart revision for the running CLI version |
| **HelmService** | `helm_service` | Wraps `helm` CLI: `repo add/update`, `install`, `upgrade`, `uninstall`, `status` |
| **DependencyService** | `dependency_service` | Verifies that `talosctl`, `nmap`, `grep`, `bash`, `awk` are available on `$PATH` |
| **HelperService** | `helper_service` | Utility functions: file checks, command piping, WSL detection, IP validation, config path resolution |
| **UiService** | `ui_service` | Interactive TUI prompts: single-select, text input, multi-select |
| **S3Service** | `s3_service` | AWS SDK v2 wrapper: create bucket, list buckets, get/put objects, enable encryption + versioning |

---

## Testability

Every service is defined as a Go interface under `interfaces/`. The `mocks/` package provides test doubles for all services. Commands in `cmd/` accept interfaces as parameters, making every command fully unit-testable without any network, filesystem, or OS dependencies.

---

## Relationship with bbe-charts

At runtime, `PackageService` fetches `library.yaml` from the [bbe-charts](https://github.com/Brains-Beyond-Expectations/bbe-charts) repository. This file lists all available Helm chart bundles alongside the minimum CLI version required to use each entry. BBE-Quest selects the most recent compatible entry and delegates installation to `HelmService`, which calls `helm install/upgrade` against the user's Kubernetes cluster context.

See [bbe-charts architecture](https://github.com/Brains-Beyond-Expectations/bbe-charts/blob/main/docs/architecture.md) for how `library.yaml` is generated.
