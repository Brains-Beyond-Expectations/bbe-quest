.PHONY: go-mod-tidy install-go-test-coverage check-coverage

GOBIN?=$$(go env GOPATH)/bin

go-mod-tidy:
	go mod tidy

install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

test: go-mod-tidy install-go-test-coverage
	go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
	${GOBIN}/go-test-coverage --config=.testcoverage.yaml $(if $(GHA),-o)
