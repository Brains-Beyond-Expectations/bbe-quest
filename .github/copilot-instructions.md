---
applyTo: "**"
---

# BBE-Quest — GitHub Copilot Instructions

## What This Project Is

BBE-Quest is a Go CLI (`bbe`) that automates the end-to-end setup and lifecycle management of a home-lab Kubernetes cluster running on Talos Linux. It handles node provisioning, cluster bootstrapping, and Helm package installation.

- **Go module**: `github.com/Brains-Beyond-Expectations/bbe-quest/cli`
- **Go version**: 1.23
- **CLI framework**: Cobra (`github.com/spf13/cobra`)
- **Testing**: `github.com/stretchr/testify` (assert + mock)
- **Config dir**: `~/.bbe/`

---

## Architecture: Commands → Interfaces → Services

All source lives under `cli/`. Cobra commands in `cmd/` are thin orchestrators. All business logic lives in `services/`. Every service is defined as a Go interface in `interfaces/` and has a `testify/mock` implementation in `mocks/`.

```
cmd/          ← Cobra command structs + init() — no logic here
interfaces/   ← One interface per service (e.g. ConfigServiceInterface)
services/     ← Concrete implementations (e.g. config_service/config_service.go)
mocks/        ← Mock implementations for unit tests (e.g. MockConfigService)
models/       ← YAML-mapped data structures (BbeConfig, LibraryEntry, etc.)
constants/    ← All constant values and error sentinels
```

---

## Non-Negotiable Conventions

### Service pattern — always required for new services

Every new service **must** follow this three-file pattern:

```
interfaces/<name>_service.go        ← interface definition
services/<name>_service/<name>_service.go  ← concrete implementation
mocks/<name>_service_mock.go        ← testify/mock mock
```

Interface naming: `<Name>ServiceInterface`  
Mock naming: `Mock<Name>Service`  
Package naming: `<name>_service` (snake_case)

### Command structure

Commands always separate the Cobra plumbing from testable logic:

```go
// Cobra entry point — construct concrete services, call the real function
var fooCmd = &cobra.Command{
    Run: func(cmd *cobra.Command, args []string) {
        err := fooCommand(foo_service.FooService{}, helper_service.HelperService{})
        if err != nil { logger.Error("", err); os.Exit(1) }
    },
}

// Testable function — accepts interfaces, never concrete types
func fooCommand(fooSvc interfaces.FooServiceInterface, helperSvc interfaces.HelperServiceInterface) error {
    // business logic here
}
```

### Tests

- Test files sit next to source files (`cmd/install_test.go` beside `cmd/install.go`).
- Only test the exported `<name>Command(...)` function, not the Cobra struct.
- All external calls (filesystem, network, OS, AWS) must be mocked — no real I/O in unit tests.
- Always call `mockFoo.AssertExpectations(t)` at the end of each test.

```go
func TestInstallCommand_Success(t *testing.T) {
    mockConfig := new(mocks.MockConfigService)
    mockConfig.On("GetBbeConfig", mock.Anything).Return(&models.BbeConfig{...}, nil)
    
    err := installCommand(mockHelper, mockUi, mockConfig, mockPackage, mockHelm)
    
    assert.NoError(t, err)
    mockConfig.AssertExpectations(t)
}
```

### Dependencies

- **Do not add new third-party Go packages** without discussion. Use stdlib or existing dependencies first.
- Existing direct deps: `aws-sdk-go-v2`, `cobra`, `testify`, `spinner`, `jsonschema`, `codename`.

### Constants and strings

- No hardcoded strings in commands or services. All constant values (file names, URLs, error messages) belong in `constants/constants.go`.
- `constants.Version` is injected at build time via `ldflags`.

---

## Key Data Structures

```go
// BbeConfig — deserialized from ~/.bbe/bbe.yaml
type BbeConfig struct {
    Bbe struct {
        Cluster struct { Name, Context string }
        Storage struct { Type string; Aws struct { BucketName string } }
        Packages []LocalPackage
    }
}

// LocalPackage — a Helm chart installed on the cluster
type LocalPackage struct { Name, Version string }

// LibraryEntry — one version window from bbe-charts/library.yaml
type LibraryEntry struct {
    MinBbeCli    string      `yaml:"min-bbe-cli"`
    ListRevision int         `yaml:"list-revision"`
    Charts       []ChartEntry
}
```

---

## Package Values Files

`PackageService` handles chart configuration automatically via `ensureValuesFile()`:

- Before `helm install/upgrade`, the function checks for `~/.bbe/<package>-values.yaml`
- If the file exists, it is re-used silently (no re-prompting on upgrade)
- If not, it fetches `values.schema.json` from bbe-charts, extracts required string leaf fields, prompts via `UiService.CreateInput()`, and writes `~/.bbe/<package>-values.yaml` (mode `0600`)
- The file path is passed to `HelmService.InstallChart`/`UpgradeChart` as the `valuesFile` parameter (appended as `-f <path>` to the `helm` command)
- Packages with no schema or no required fields receive `valuesFile = ""` and no `-f` flag is added

---

## Services at a Glance

| Interface | Concrete package | Purpose |
|---|---|---|
| `ConfigServiceInterface` | `config_service` | Read/write `bbe.yaml`, sync with S3 |
| `TalosServiceInterface` | `talos_service` | Wraps `talosctl` CLI |
| `ImageServiceInterface` | `image_service` | Downloads Talos factory images |
| `IpFinderServiceInterface` | `ipfinder_service` | Node discovery via `nmap` |
| `PackageServiceInterface` | `package_service` | Fetches `library.yaml`, prompts for required chart values, delegates to HelmService |
| `HelmServiceInterface` | `helm_service` | Wraps `helm` CLI; `InstallChart`/`UpgradeChart` accept `valuesFile string` (appended as `-f`) |
| `DependencyServiceInterface` | `dependency_service` | Checks `$PATH` for required tools |
| `HelperServiceInterface` | `helper_service` | Filesystem, WSL, IP utilities |
| `UiServiceInterface` | `ui_service` | TUI prompts (select, text, multi-select) |
| `S3ServiceInterface` | `s3_service` | AWS S3 CRUD + bucket setup |

---

## Relationship with bbe-charts

`PackageService` fetches `library.yaml` from `constants.BbeLibraryUrl` at runtime. It selects the entry with the highest `list-revision` where `min-bbe-cli ≤ constants.Version`, then passes chart metadata to `HelmService` for installation on the cluster.

Documentation lives in `docs/`. Each command has a corresponding `docs/commands/<name>.md`.
