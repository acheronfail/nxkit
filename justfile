go_dir := 'go'
nxkit_image := 'nxkit'
git_version := `
tag="$(git describe --tags --exact-match --match 'v*' 2>/dev/null || true)"

if printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  printf '%s' "${tag#v}"
else
  sha="$(git rev-parse --short HEAD 2>/dev/null || printf unknown)"
  printf '0.0.0+%s' "$sha"
fi
`
version := env_var_or_default("NXKIT_VERSION", git_version)
buildVersion := env_var_or_default("NXKIT_BUILD_VERSION", "1")
goLdflags := "-X github.com/acheronfail/nxkit/gui.packageVersion=" + version
goPackageLdflags := goLdflags + " -X github.com/acheronfail/nxkit/gui.packagedBuild=true"
export NXKIT_PACKAGE_VERSION := version
export NXKIT_BUILD_VERSION := buildVersion
export NXKIT_GO_PACKAGE_LDFLAGS := goPackageLdflags
export CGO_CFLAGS := env_var_or_default("CGO_CFLAGS", "-Wno-deprecated-declarations")

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

create-nand *args:
    cd "{{ go_dir }}" && GOCACHE="${GOCACHE:-$PWD/../.gocache}" go run ./cmd/createnand {{ args }}

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
    cd "{{ go_dir }}" && go build -ldflags '{{ goLdflags }}' -o nxkit main.go

[linux]
package:
    cd "{{ go_dir }}" && bash ./scripts/package-linux.sh

[macos]
package:
    cd "{{ go_dir }}" && bash ./scripts/package-macos.sh

[windows]
package:
    cd "{{ go_dir }}" && bash ./scripts/package-windows.sh

run *args:
    cd "{{ go_dir }}" && go run -ldflags '{{ goLdflags }}' main.go {{ args }}

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
