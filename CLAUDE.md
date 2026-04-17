# BBE-Quest — AI Agent Context

## Project Overview

BBE-Quest (`bbe`) is a Go CLI tool that automates end-to-end setup and lifecycle management of a home-lab Kubernetes cluster running on [Talos Linux](https://www.talos.dev/). It provisions nodes (Intel NUC and Raspberry Pi 4), bootstraps Talos clusters, and installs/upgrades Helm chart bundles from the companion [bbe-charts](https://github.com/Brains-Beyond-Expectations/bbe-charts) repository.

Config lives in `~/.bbe/`. Charts are fetched at runtime from `https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-charts/main/library.yaml`.

---

## Repository Layout

```
cli/                             ← Go module root (all code lives here)
├── main.go                      ← Entry point — calls cmd.Execute()
├── go.mod                       ← Module: github.com/Brains-Beyond-Expectations/bbe-quest/cli
├── Makefile
├── cmd/                         ← Cobra command definitions (thin orchestrators only)
│   ├── cmd.go                   ← Root command + Execute()
│   ├── install.go / install_test.go
│   ├── setup.go   / setup_test.go
│   ├── upgrade.go / upgrade_test.go
│   ├── config.go  / config_test.go
│   └── version.go / version_test.go
├── constants/
│   └── constants.go             ← Version (ldflags), file names, BbeLibraryUrl
├── interfaces/                  ← One interface file per service
│   ├── config_service.go
│   ├── dependency_service.go
│   ├── helm_service.go
│   ├── helper_service.go
│   ├── image_service.go
│   ├── ipfinder_service.go
│   ├── packages_service.go
│   ├── s3_service.go
│   ├── talos_service.go
│   └── ui_service.go
├── models/                      ← YAML-mapped data structures
│   ├── bbe_config.go            ← BbeConfig, LocalPackage
│   ├── bbe_library.go           ← Library, LibraryEntry, ChartEntry
│   ├── node_type.go             ← NodeType (hardware targets)
│   └── talos.go                 ← TalosMachineConfig
├── services/                    ← Business logic (one package per service)
│   ├── config_service/
│   ├── dependency_service/
│   ├── helm_service/
│   ├── helper_service/
│   ├── image_service/
│   ├── ipfinder_service/
│   ├── package_service/
│   ├── s3_service/
│   ├── talos_service/
│   └── ui_service/
├── mocks/                       ← testify/mock implementations (one per service)
│   ├── config_service_mock.go
│   ├── dependency_service_mock.go
│   └── ...
└── misc/
    └── logger/                  ← Structured logging helpers
```

---

## Essential Commands

All commands run from `cli/`:

```bash
# Run any CLI command in development
go run main.go <command>
go run main.go version
go run main.go install
go run main.go setup

# Run all tests (with race detection and coverage)
make test
# equivalent: go test ./... -race -covermode=atomic -coverprofile=coverage.out

# Tidy modules
make go-mod-tidy

# Build binary
make build
```

---

## Architecture: The Service Pattern

Every piece of business logic is a **service**. The pattern is:

1. **Interface** — `interfaces/<name>_service.go`  
   Defines the contract. Commands depend only on the interface, never the concrete type.

2. **Implementation** — `services/<name>_service/<name>_service.go`  
   Concrete struct that implements the interface.

3. **Mock** — `mocks/<name>_service_mock.go`  
   `testify/mock`-based mock for unit tests. Must implement the interface exactly.

### Example: a new `FooService`

```go
// interfaces/foo_service.go
package interfaces

type FooServiceInterface interface {
    DoThing(input string) (string, error)
}
```

```go
// services/foo_service/foo_service.go
package foo_service

type FooService struct{}

func (s FooService) DoThing(input string) (string, error) {
    // implementation
}
```

```go
// mocks/foo_service_mock.go
package mocks

import (
    "github.com/stretchr/testify/mock"
)

type MockFooService struct {
    mock.Mock
}

func (m *MockFooService) DoThing(input string) (string, error) {
    args := m.Called(input)
    return args.String(0), args.Error(1)
}
```

Commands in `cmd/` accept interfaces as parameters and construct concrete services in their Cobra `Run` function only:

```go
var fooCmd = &cobra.Command{
    Run: func(cmd *cobra.Command, args []string) {
        err := fooCommand(foo_service.FooService{}, helper_service.HelperService{})
        // ...
    },
}

func fooCommand(fooService interfaces.FooServiceInterface, helperService interfaces.HelperServiceInterface) error {
    // testable business logic
}
```

---

## Coding Conventions

### Non-negotiable rules

- **All external dependencies must be mocked in tests.** No real filesystem, network, process, or AWS calls in unit tests.
- **Every new service requires an interface + mock.** A service without an interface cannot be injected and cannot be tested.
- **No new third-party dependencies without explicit discussion.** Evaluate whether stdlib or an existing dependency can solve the problem first.

### Testing

- Use `github.com/stretchr/testify/mock` for mocks and `github.com/stretchr/testify/assert` for assertions.
- Test files live alongside source files (`cmd/install_test.go` next to `cmd/install.go`).
- Test the exported command function (e.g., `installCommand(...)`) not the Cobra command itself.
- Always call `mock.AssertExpectations(t)` at the end of each test.

### Error handling

- Return `error` from all functions that can fail. Do not panic except in scenarios that are truly unrecoverable programmer errors.
- Log errors with `logger.Error(...)`. Log informational messages with `logger.Info(...)`.

### Naming

- Service interfaces: `<Name>ServiceInterface` (e.g., `HelmServiceInterface`)
- Mock structs: `Mock<Name>Service` (e.g., `MockHelmService`)
- Concrete service packages: `<name>_service` (snake_case package name)

### Constants and URLs

- All constant values belong in `constants/constants.go`. Do not hardcode strings in commands or services.
- `constants.Version` is injected at build time via `ldflags`. It defaults to `"development"`.

---

## Key Data Models

```go
// ~/.bbe/bbe.yaml — master config
type BbeConfig struct {
    Bbe struct {
        Cluster struct {
            Name    string
            Context string
        }
        Storage struct {
            Type string       // "local" or "aws"
            Aws  struct {
                BucketName string `yaml:"bucket_name"`
            }
        }
        Packages []LocalPackage
    }
}

// A package installed via `bbe install`
type LocalPackage struct {
    Name    string
    Version string
}

// Entry from library.yaml fetched from bbe-charts
type LibraryEntry struct {
    MinBbeCli    string       `yaml:"min-bbe-cli"`
    ListRevision int          `yaml:"list-revision"`
    Charts       []ChartEntry
}
```

---

## Package Values Files

When a chart has a `values.schema.json` with required fields, `PackageService` handles configuration automatically:

1. Before calling `helm install/upgrade`, `ensureValuesFile()` is called.
2. It checks whether `~/.bbe/<package>-values.yaml` already exists. If so, it is re-used silently.
3. If not, it fetches `values.schema.json` from bbe-charts, walks it to find required string leaf fields, and prompts the user for each value via `UiService.CreateInput()`.
4. The values are written as nested YAML to `~/.bbe/<package>-values.yaml` (mode `0600`).
5. The file path is passed to `helm install/upgrade` as `-f <valuesFile>`.

Packages with no `values.schema.json`, or schemas with no required fields (e.g. `bbe-media`), skip this flow entirely and pass `valuesFile = ""`.

To reconfigure a package's values, delete its file and re-run `bbe install`:
```bash
rm ~/.bbe/bbe-networking-values.yaml
bbe install
```

---

## Relationship with bbe-charts

- `constants.BbeLibraryUrl` → `https://raw.githubusercontent.com/Brains-Beyond-Expectations/bbe-charts/main/library.yaml`
- `PackageService.GetAll()` fetches this URL and returns the most recent `LibraryEntry` compatible with `constants.Version`.
- `HelmService` then calls `helm repo add` + `helm install/upgrade` for each chart in the entry.
- The two repos **share identical model types** (`Library`, `LibraryEntry`, `ChartEntry`) — they are deliberately duplicated, not extracted into a shared module.

---

## Adding a New Command

1. Create `cmd/<name>.go` with the Cobra command definition.
2. Define the testable function: `func <name>Command(dep1 interfaces.X, dep2 interfaces.Y) error`.
3. Register with `rootCmd.AddCommand(...)` in an `init()` function.
4. Create `cmd/<name>_test.go` using mocks.
5. Document the command in `docs/commands/<name>.md`.

## Adding a New Service

1. Define the interface in `interfaces/<name>_service.go`.
2. Implement it in `services/<name>_service/<name>_service.go`.
3. Add a mock in `mocks/<name>_service_mock.go`.
4. Inject the interface into any command that needs it.
