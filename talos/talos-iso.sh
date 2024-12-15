#!/bin/bash

BLUE='\033[0;34m'
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${BLUE}Creating Talos ISO${NC}"

if ! crane version &>/dev/null; then
    echo -e "${RED}crane is not installed. Please install it${NC}"
    exit 1
fi

if ! docker ps &>/dev/null; then
    echo -e "${RED}Docker is not running. Please start it${NC}"
    exit 1
fi

extensions=$(crane export ghcr.io/siderolabs/extensions:v1.8.0 | tar x -O image-digests | grep -E 'intel-ucode:|gvisor:|iscsi-tools')

docker run --rm -t -v $PWD/talos/_out:/out ghcr.io/siderolabs/imager:v1.8.0 iso $(echo "$extensions" | awk '{print "--system-extension-image " $1}' | tr '\n' ' ')

echo -e "${GREEN}Talos ISO saved to talos/_out${NC}"
