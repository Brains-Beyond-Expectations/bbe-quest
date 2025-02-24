.PHONY: go-mod-tidy test all clean

GOBIN?=$$(go env GOPATH)/bin

go-mod-tidy:
	go mod tidy

test: go-mod-tidy
	go test ./... -race -covermode=atomic -coverprofile=coverage.out
