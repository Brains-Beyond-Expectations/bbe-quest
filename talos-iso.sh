#!/bin/bash

extensions=$(crane export ghcr.io/siderolabs/extensions:v1.8.0 | tar x -O image-digests | grep -E 'intel-ucode:|gvisor:|iscsi-tools')

extension_args=$(echo "$extensions" | awk '{print "--system-extension-image " $1}' | tr '\n' ' ')

docker run --rm -t -v $PWD/_out:/out ghcr.io/siderolabs/imager:v1.8.0 iso $extension_args --extra-kernel-arg net.ifnames=0 --extra-kernel-arg=-console --extra-kernel-arg=console=ttyS1
