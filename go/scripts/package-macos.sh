#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
go_dir="$(cd "${script_dir}/.." && pwd)"
cd "$go_dir"
source "${script_dir}/package-common.sh"

nxkit_package_setup

arch="$(go env GOARCH)"
platform="macos"
binary="$(nxkit_binary_name "$platform" "$arch")"
version="$(nxkit_package_version)"
build_version="$(nxkit_build_version)"

nxkit_build_binary "$binary"

app_dir="dist/NXKit.app"
plist="${app_dir}/Contents/Info.plist"
resources_dir="${app_dir}/Contents/Resources"
icon_source="resources/nxkit-icon-subtle.png"
icon_source_256="resources/nxkit-icon-subtle-256.png"
icon_source_svg="resources/nxkit-icon-subtle.svg"
icon_source_icns="resources/NXKit.icns"

rm -rf "$app_dir"
mkdir -p "${app_dir}/Contents/MacOS" "$resources_dir"
cp "dist/${binary}" "${app_dir}/Contents/MacOS/${binary}"
chmod +x "${app_dir}/Contents/MacOS/${binary}"

cp "$icon_source" "${resources_dir}/nxkit-icon-subtle.png"
cp "$icon_source_256" "${resources_dir}/nxkit-icon-subtle-256.png"
cp "$icon_source_svg" "${resources_dir}/nxkit-icon-subtle.svg"
cp "$icon_source_icns" "${resources_dir}/NXKit.icns"

plutil -create xml1 "$plist"
/usr/libexec/PlistBuddy -c "Add :CFBundleName string NXKit" "$plist"
/usr/libexec/PlistBuddy -c "Add :CFBundleDisplayName string NXKit" "$plist"
/usr/libexec/PlistBuddy -c "Add :CFBundleExecutable string ${binary}" "$plist"
/usr/libexec/PlistBuddy -c "Add :CFBundleIconFile string NXKit.icns" "$plist"
/usr/libexec/PlistBuddy -c "Add :CFBundleIdentifier string fail.acheron.nxkit" "$plist"
/usr/libexec/PlistBuddy -c "Add :CFBundlePackageType string APPL" "$plist"
/usr/libexec/PlistBuddy -c "Add :CFBundleShortVersionString string ${version}" "$plist"
/usr/libexec/PlistBuddy -c "Add :CFBundleVersion string ${build_version}" "$plist"
/usr/libexec/PlistBuddy -c "Add :LSApplicationCategoryType string public.app-category.utilities" "$plist"
/usr/libexec/PlistBuddy -c "Add :NSHighResolutionCapable bool true" "$plist"

tar -czf "dist/NXKit-${platform}-${arch}.app.tar.gz" -C dist NXKit.app
echo "Packaged dist/NXKit-${platform}-${arch}.app.tar.gz"
