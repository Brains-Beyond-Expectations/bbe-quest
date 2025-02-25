.PHONY: all clean go-mod-tidy test build

GOBIN?=$$(go env GOPATH)/bin

# Default target that lists all available commands and their descriptions
all:
	@echo "Available commands:"
	@echo "  go-mod-tidy    - Tidy up Go modules"
	@echo "  test           - Run tests with race detection and coverage"
	@echo "  build          - Builds the project"
	@echo "  clean          - Remove build artifacts"

go-mod-tidy:
	go mod tidy

test: go-mod-tidy
	go test ./... -race -covermode=atomic -coverprofile=coverage.out

build: go-mod-tidy
	go build

clean:
	rm -rf _out/
	rm -f coverage.out
	rm -f cover.out