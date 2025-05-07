#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE[0]}")"

image_name="nxkit-fs-fixtures"

docker build -t "${image_name}" .
docker run --rm --user "$(id -u):$(id -g)" -i -v "${PWD}:/mnt" "${image_name}"
