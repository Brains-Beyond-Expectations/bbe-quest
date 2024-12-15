#!/bin/bash

VERSION=$(gh release list --limit 1 --json tagName -q '.[0].tagName')

mkdir -p tmp

gh release download $VERSION --repo nicolajv/bbe-quest --pattern "*linux-amd64.tar.gz" --dir ./tmp

tar -xvf ./tmp/bbe-$VERSION-linux-amd64.tar.gz

rm -rf tmp

sudo mv bbe /usr/local/bin
