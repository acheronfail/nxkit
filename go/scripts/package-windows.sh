#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
go_dir="$(cd "${script_dir}/.." && pwd)"
cd "$go_dir"
source "${script_dir}/package-common.sh"

nxkit_package_setup
trap 'rm -f resource_windows_*.syso' EXIT

arch="$(go env GOARCH)"
binary="$(nxkit_binary_name windows "$arch" .exe)"
version="$(nxkit_package_version)"
build_version="$(nxkit_build_version)"
icon_path="resources/NXKit.ico"

version_core="${version%%[-+]*}"
IFS=. read -r ver_major ver_minor ver_patch <<< "$version_core"
for part in ver_major ver_minor ver_patch; do
  if ! [[ "${!part:-}" =~ ^[0-9]+$ ]]; then
    printf -v "$part" '%s' "0"
  fi
done

arch_flag=()
case "$arch" in
  amd64)
    arch_flag=(-64)
    ;;
  arm)
    arch_flag=(-arm)
    ;;
  arm64)
    arch_flag=(-arm -64)
    ;;
esac

rm -f resource_windows_*.syso
(unset GOOS GOARCH; go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0 \
  -skip-versioninfo \
  -icon "$icon_path" \
  -application-icon "$icon_path" \
  -o "resource_windows_${arch}.syso" \
  "${arch_flag[@]}" \
  -description "NXKit" \
  -product-name "NXKit" \
  -internal-name "${binary}" \
  -original-name "${binary}" \
  -file-version "$version" \
  -product-version "$version" \
  -ver-major "$ver_major" \
  -ver-minor "$ver_minor" \
  -ver-patch "$ver_patch" \
  -ver-build "$build_version" \
  -product-ver-major "$ver_major" \
  -product-ver-minor "$ver_minor" \
  -product-ver-patch "$ver_patch" \
  -product-ver-build "$build_version")

NXKIT_GO_PACKAGE_LDFLAGS="${NXKIT_GO_PACKAGE_LDFLAGS} -H=windowsgui" nxkit_build_binary "$binary"
echo "Built dist/${binary}"
