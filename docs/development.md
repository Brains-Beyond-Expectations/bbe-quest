# Local Development

## Requirements

- **Go 1.23+**
- `talosctl`, `nmap`, `helm` (for integration testing against a real cluster)

---

## Setup

Clone the repository and change into the CLI directory:

```bash
git clone https://github.com/Brains-Beyond-Expectations/bbe-quest.git
cd bbe-quest/cli
```

---

## Running Commands Locally

Use `go run` to run any CLI command directly:

```bash
go run main.go <command>
# examples:
go run main.go version
go run main.go install
```

---

## Running Tests

```bash
make test
```

This runs all unit tests with coverage output. Tests use mocks from `mocks/` and do not require a live cluster or network.

To run tests for a single package:

```bash
go test ./cmd/...
go test ./services/config_service/...
```

---

## Project Conventions

### Adding a New Service

1. Define the interface in `interfaces/<name>_service.go`
2. Implement it in `services/<name>_service/`
3. Add a mock in `mocks/<name>_service_mock.go`
4. Inject the interface into command parameters in `cmd/`

### Adding a New Command

1. Create `cmd/<name>.go` with the Cobra command definition
2. Accept all external dependencies as interface parameters
3. Register the command in `cmd/cmd.go`
4. Add tests in `cmd/<name>_test.go` using mocks

---

## Build

The version is injected at build time:

```bash
go build -ldflags "-X github.com/Brains-Beyond-Expectations/bbe-quest/constants.Version=<version>" -o bbe ./main.go
```

The `Makefile` automates this.

---

## Code Coverage

Coverage reports are uploaded to [Codecov](https://codecov.io/gh/Brains-Beyond-Expectations/bbe-quest). The `codecov.yaml` at the repository root configures coverage thresholds.
