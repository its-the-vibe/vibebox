#!/usr/bin/env bash

set -euo pipefail

# Fetch latest Go version
LATEST_GO_VERSION=$(curl -s 'https://go.dev/VERSION?m=text' | head -n 1)

echo "Beginning download of latest Go version: ${LATEST_GO_VERSION}"

# Download the archive
TARBALL="${LATEST_GO_VERSION}.linux-arm64.tar.gz"
curl -sSL "https://go.dev/dl/${TARBALL}" -o "${TARBALL}"

echo "Beginning installation of latest Go..."

# Remove previous installation and extract new version
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "${TARBALL}"

# Clean up local archive
rm -f "${TARBALL}"

/usr/local/go/bin/go version

echo "Finished installing latest Go."
