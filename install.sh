#!/bin/bash

set -e

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

LATEST_URL=$(curl -s https://api.github.com/repos/Brains-Beyond-Expectations/bbe-quest/releases/latest | grep "browser_download_url.*${OS}_${ARCH}.tar.gz\"" | cut -d '"' -f 4)

TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

curl -L "$LATEST_URL" -o bbe-cli.tar.gz

tar -xzf bbe-cli.tar.gz
sudo mv bbe /usr/local/bin

bbe version
