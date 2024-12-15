build-cli:
	cd cli && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../_cli/amd64/bbe -ldflags "-X 'github.com/nicolajv/bbe-quest/constants.Version=v1.0.0'"

talos-iso:
	bash talos/talos-iso.sh
