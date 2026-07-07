nxkit_package_setup() {
  export GOCACHE="${GOCACHE:-$PWD/../.gocache}"
  mkdir -p dist
}

nxkit_package_version() {
  printf '%s' "${NXKIT_PACKAGE_VERSION:-0.0.0}"
}

nxkit_build_version() {
  local build_version="${NXKIT_BUILD_VERSION:-1}"
  if ! [[ "$build_version" =~ ^[0-9]+$ ]]; then
    build_version="1"
  fi
  printf '%s' "$build_version"
}

nxkit_binary_name() {
  local platform="$1"
  local arch="$2"
  local extension="${3:-}"
  printf 'nxkit-%s-%s%s' "$platform" "$arch" "$extension"
}

nxkit_build_binary() {
  local binary="$1"
  go build -trimpath -ldflags "-X github.com/acheronfail/nxkit/gui.packagedBuild=true" -o "dist/${binary}" main.go
}
