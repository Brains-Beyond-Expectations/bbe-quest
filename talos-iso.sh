#!/bin/bash

if ! crane version &>/dev/null; then
    echo "crane is not installed. Please install it"
    exit 1
fi

extensions=$(crane export ghcr.io/siderolabs/extensions:v1.8.0 | tar x -O image-digests | grep -E 'intel-ucode:|gvisor:|iscsi-tools')

docker run --rm -t -v $PWD/_out:/out ghcr.io/siderolabs/imager:v1.8.0 iso $(echo "$extensions" | awk '{print "--system-extension-image " $1}' | tr '\n' ' ')
