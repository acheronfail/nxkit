#!/usr/bin/env bash

set -euo pipefail

image_name="nxkit-fs-fixtures"

cd "$(dirname "${BASH_SOURCE[0]}")"
echo "Preparing fixtures..."

docker build -t "${image_name}" . 2>/dev/null
docker run --rm --user "$(id -u):$(id -g)" -i -v "${PWD}:/mnt" "${image_name}"
