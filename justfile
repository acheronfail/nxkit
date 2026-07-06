go_dir := 'go'
nxkit_image := 'nxkit'

_default:
    just -l

#
# Go
#

setup:
    cd "{{ go_dir }}" && go mod tidy
    go install github.com/cespare/reflex@latest
    just fixtures

fixtures: test-keys
    cd "{{ go_dir }}" && bash ./lib/fat/testdata/create.sh
    cd "{{ go_dir }}" && make -C ./lib/romfs/testdata

test-keys:
    cd "{{ go_dir }}" && GOCACHE="${GOCACHE:-$PWD/../.gocache}" go run ./cmd/testkeys testdata/prod.keys .data/prod.keys prod.keys

test n *args: fixtures
    cd "{{ go_dir }}" && FAT={{ n }} go test github.com/acheronfail/nxkit/... {{ args }}
    cd "{{ go_dir }}" && FAT={{ n }} go test github.com/acheronfail/nxkit/... {{ args }}

testw n *args:
    cd "{{ go_dir }}" && reflex -d none -sr '\.go$' -- sh -c 'cd .. && just test {{ n }} {{ args }}'

test-all: fixtures
    cd "{{ go_dir }}" && FAT=12 go test github.com/acheronfail/nxkit/...
    cd "{{ go_dir }}" && FAT=12 go test github.com/acheronfail/nxkit/...
    cd "{{ go_dir }}" && FAT=16 go test github.com/acheronfail/nxkit/...
    cd "{{ go_dir }}" && FAT=16 go test github.com/acheronfail/nxkit/...
    cd "{{ go_dir }}" && FAT=32 go test github.com/acheronfail/nxkit/...
    cd "{{ go_dir }}" && FAT=32 go test github.com/acheronfail/nxkit/...

bench:
    cd "{{ go_dir }}" && go test -bench=. github.com/acheronfail/nxkit/...

build:
    cd "{{ go_dir }}" && go build -o nxkit main.go

package:
    #!/usr/bin/env bash
    set -euo pipefail
    cd go

    os="$(go env GOOS)"
    arch="$(go env GOARCH)"
    platform="$os"
    if [[ "$os" == "darwin" ]]; then
      platform="macos"
    fi

    binary="nxkit-${platform}-${arch}"
    if [[ "$os" == "windows" ]]; then
      binary="${binary}.exe"
    fi

    mkdir -p dist
    go build -trimpath -o "dist/${binary}" main.go

    if [[ "$os" != "darwin" ]]; then
      echo "Built dist/${binary}"
      exit 0
    fi

    app_dir="dist/NXKit.app"
    plist="${app_dir}/Contents/Info.plist"
    resources_dir="${app_dir}/Contents/Resources"
    icon_source="resources/nxkit-icon-subtle.png"
    icon_source_256="resources/nxkit-icon-subtle-256.png"
    icon_source_svg="resources/nxkit-icon-subtle.svg"
    iconset="${resources_dir}/NXKit.iconset"

    rm -rf "$app_dir"
    mkdir -p "${app_dir}/Contents/MacOS" "$resources_dir" "$iconset"
    cp "dist/${binary}" "${app_dir}/Contents/MacOS/${binary}"
    chmod +x "${app_dir}/Contents/MacOS/${binary}"

    cp "$icon_source" "${resources_dir}/nxkit-icon-subtle.png"
    cp "$icon_source_256" "${resources_dir}/nxkit-icon-subtle-256.png"
    cp "$icon_source_svg" "${resources_dir}/nxkit-icon-subtle.svg"
    sips -z 16 16 "$icon_source" --out "${iconset}/icon_16x16.png" >/dev/null
    sips -z 32 32 "$icon_source" --out "${iconset}/icon_16x16@2x.png" >/dev/null
    sips -z 32 32 "$icon_source" --out "${iconset}/icon_32x32.png" >/dev/null
    sips -z 64 64 "$icon_source" --out "${iconset}/icon_32x32@2x.png" >/dev/null
    sips -z 128 128 "$icon_source" --out "${iconset}/icon_128x128.png" >/dev/null
    cp "$icon_source_256" "${iconset}/icon_128x128@2x.png"
    cp "$icon_source_256" "${iconset}/icon_256x256.png"
    sips -z 512 512 "$icon_source" --out "${iconset}/icon_256x256@2x.png" >/dev/null
    sips -z 512 512 "$icon_source" --out "${iconset}/icon_512x512.png" >/dev/null
    cp "$icon_source" "${iconset}/icon_512x512@2x.png"
    icon_file=""
    if iconutil --convert icns "$iconset" --output "${resources_dir}/NXKit.icns"; then
      icon_file="NXKit.icns"
    else
      echo "warning: failed to create NXKit.icns; packaging app without a bundle icon" >&2
    fi
    rm -rf "$iconset"

    plutil -create xml1 "$plist"
    /usr/libexec/PlistBuddy -c "Add :CFBundleName string NXKit" "$plist"
    /usr/libexec/PlistBuddy -c "Add :CFBundleDisplayName string NXKit" "$plist"
    /usr/libexec/PlistBuddy -c "Add :CFBundleExecutable string ${binary}" "$plist"
    if [[ -n "$icon_file" ]]; then
      /usr/libexec/PlistBuddy -c "Add :CFBundleIconFile string ${icon_file}" "$plist"
    fi
    /usr/libexec/PlistBuddy -c "Add :CFBundleIdentifier string fail.acheron.nxkit" "$plist"
    /usr/libexec/PlistBuddy -c "Add :CFBundlePackageType string APPL" "$plist"
    /usr/libexec/PlistBuddy -c "Add :CFBundleShortVersionString string 0.0.1" "$plist"
    /usr/libexec/PlistBuddy -c "Add :CFBundleVersion string ${BUILD_NUMBER:-1}" "$plist"
    /usr/libexec/PlistBuddy -c "Add :LSApplicationCategoryType string public.app-category.utilities" "$plist"
    /usr/libexec/PlistBuddy -c "Add :NSHighResolutionCapable bool true" "$plist"

    tar -czf "dist/NXKit-${platform}-${arch}.app.tar.gz" -C dist NXKit.app
    echo "Packaged dist/NXKit-${platform}-${arch}.app.tar.gz"

run *args:
    cd "{{ go_dir }}" && go run main.go {{ args }}

dev *args:
    cd "{{ go_dir }}" && reflex -d none -sr '\.go$' -- sh -c 'cd .. && just run {{ args }}'

#
# Vendor
#

_nxkit_image:
    docker build --platform linux/amd64 --tag {{ nxkit_image }} vendor/docker

# rebuilds all vendor dependencies
vendor: _nxkit_image
    for c in $(just --summary | xargs -n1 | grep vendor-); do just $c; done

vendor-nro: _nxkit_image
    docker run --rm -v "$PWD/vendor/Forwarder-Mod:/src" {{ nxkit_image }} bash -c '(cd /src; make clean; make all)'
    cp vendor/Forwarder-Mod/hbl.nso go/lib/hacbrewpack/assets/main.nso
    cp vendor/Forwarder-Mod/hbl.npdm go/lib/hacbrewpack/assets/main.npdm

vendor-hacbrewpack: _nxkit_image
    docker run --rm -v "$PWD/vendor/hacbrewpack:/src" {{ nxkit_image }} bash -c '(cd /src; make clean_full; make)'

go-assets: vendor-nro

fetch-titles:
    cd legacy && npm exec tsx ../vendor/tinfoil/update.ts
