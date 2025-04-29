#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE[0]}")"

image_name="nxkit-fs-fixtures"

docker build -t "${image_name}" .
cat <<"EOF" | docker run --rm -i -v "${PWD}:/mnt" "${image_name}"

set -euo pipefail

echo "Creating fixtures..."
for n in $(echo "12 16 32"); do
  out="/mnt/fat${n}"
  mkdir -p ${out}
  dd if=/dev/zero of=${out}/disk.img bs=1M count=$(( n + 2 ))
  mkfs.vfat -v -F ${n} ${out}/disk.img
done

EOF
