#!/bin/bash

if ! crane version &>/dev/null; then
    echo "crane is not installed. Please install it"
    exit 1
fi

read -p "Do you want secureboot? [Y/n] " sb_response
sb_response=${sb_response:-Y}
use_secureboot=false

if [[ ${sb_response,,} =~ ^y ]]; then
    use_secureboot=true
fi

extensions=$(crane export ghcr.io/siderolabs/extensions:v1.8.0 | tar x -O image-digests | grep -E 'intel-ucode:|gvisor:|iscsi-tools')

if [[ $use_secureboot == true ]]; then
    talosctl gen secureboot uki --common-name "SecureBoot Key"
    talosctl gen secureboot pcr
    iso_type="secureboot-iso"
    secure_boot_volume="-v $PWD/_out:/secureboot:ro"
else
    iso_type="iso"
fi

docker run --rm -t $secure_boot_volume -v $PWD/_out:/out ghcr.io/siderolabs/imager:v1.8.0 $iso_type $(echo "$extensions" | awk '{print "--system-extension-image " $1}' | tr '\n' ' ') --extra-kernel-arg net.ifnames=0 --extra-kernel-arg=-console --extra-kernel-arg=console=ttyS1
