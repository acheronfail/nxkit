#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
go_dir="$(cd "${script_dir}/.." && pwd)"
cd "$go_dir"
source "${script_dir}/package-common.sh"

nxkit_package_setup

arch="$(go env GOARCH)"
binary="$(nxkit_binary_name linux "$arch")"

nxkit_build_binary "$binary"

package_dir="dist/${binary}-package"
rm -rf "$package_dir"
mkdir -p "$package_dir"
cp "dist/${binary}" "${package_dir}/${binary}"
chmod +x "${package_dir}/${binary}"
cp "resources/nxkit-icon-subtle-256.png" "${package_dir}/nxkit-icon-subtle.png"
cat > "${package_dir}/NXKit.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=NXKit
Comment=Nintendo Switch file utility
Exec=${binary}
Icon=nxkit-icon-subtle
Terminal=false
Categories=Utility;
EOF

rm -f "dist/${binary}.zip"
(
  cd "$package_dir"
  zip -q -r "../${binary}.zip" .
)
echo "Packaged dist/${binary}.zip"
