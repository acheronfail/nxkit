cyan := '\033[0;36m'
red := '\033[0;31m'
reset := '\033[0m'

_default:
  just -l

@_check CMD MSG="":
    if ! command -v {{CMD}} >/dev/null 2>&1 /dev/null; then echo "{{CMD}} is required! {{MSG}}"; exit 1; fi

# set up the local repository for development
setup:
  @echo "{{cyan}}Installing node_modules...{{reset}}"
  npm install --force

  @echo "{{cyan}}Setting up git hooks...{{reset}}"
  @echo "#!/usr/bin/env bash" > .git/hooks/pre-commit
  @echo "just pre-commit" >> .git/hooks/pre-commit
  @chmod +x .git/hooks/pre-commit

  @echo "{{cyan}}Checking for required tools...{{reset}}"
  @just _check "gcc"
  @just _check "make"
  @just _check "emcc" "See https://emscripten.org/docs/getting_started/downloads.html for installation instructions"
  @just _check "dkp-pacman" "See https://devkitpro.org/wiki/Getting_Started for installation instructions"

  @echo "{{cyan}}Checking for devkitpro switch packages...{{reset}}"
  @if ! dkp-pacman -Qg switch-dev | cut -d' ' -f2 | xargs -n1 sh -c 'dkp-pacman -Qi $0 >/dev/null || exit 255'; then \
    echo "{{red}}Failed to find devkitpro switch-dev package, installing now...{{reset}}" \
    sudo dkp-pacman -Sy switch-dev; \
  fi

  @echo "{{cyan}}Building vendor dependencies...{{reset}}"
  just vendor

  @echo -n "{{cyan}}Do you want to create a fake nand dump?{{reset}} (y/N): "
  @read ans; if [[ $ans = *y* ]]; then just create-nand; just rebuild-electron; fi

#
# Dev Scripts
#

# start the app in dev mode
dev *args: rebuild-electron
  npm start -- -- {{args}}

# rebuild native modules to work with electron
rebuild-electron:
  cd src/node/nand/xtsn && npm run clean
  npm exec electron-rebuild -- --module-dir src/node/nand/xtsn
# rebuild native modules to work with node
rebuild-node:
  cd src/node/nand/xtsn && npm rebuild

# runs all tests and checks
test-all: rebuild-node
  npm run test

# runs vitest in watch mode
test *ARGS:
  npm run test:vitest -- {{ARGS}}

# runs vitest in bench mode
bench *ARGS: rebuild-node
  npm exec vitest -- bench --run -- {{ARGS}}

# runs a benchmark copying a 100M file into an Xtsn-enxrypted FAT32 disk image, outputs a cpuprofile
bench100m: rebuild-node
  @mkdir -p scripts/build/Release
  @cp src/node/nand/xtsn/build/Release/xtsn.node scripts/build/Release/xtsn.node
  @cp node_modules/js-fatfs/dist/fatfs.wasm scripts/
  npx esbuild --bundle --platform=node --format=esm scripts/bench100m.ts --outfile=scripts/bench100m.js
  node --cpu-prof scripts/bench100m.js

# formats all code
format:
  npm run format

# creates a fake NAND dump for testing
create-nand *ARGS: rebuild-node
  npm exec tsx scripts/create-fake-nand.ts -- {{ARGS}}

#
# Vendor
#

# builds all vendor dependencies
vendor:
  for c in $(just --summary | xargs -n1 | grep vendor-); do just $c; done

vendor-nro:
  cd vendor/Forwarder-Mod && (make clean; make all)
vendor-hacbrewpack:
  cd vendor/hacbrewpack && (make clean_full; make)

fetch-titles:
  npm exec tsx vendor/tinfoil/update.ts

#
# Release
#

publish-xtsn:
  cd src/node/nand/xtsn && npm run prepublish && npm publish

#
# Hooks
#

# hook that's run pre-commit
pre-commit:
  #!/usr/bin/env bash
  set -uo pipefail

  git diff --exit-code >/dev/null
  needs_save=$?

  set -e

  saved="precommit.diff"
  if [ $needs_save -ne 0 ]; then
    git diff > "$saved"
    git apply -R "$saved"
  fi
  just format
  just test-all
  if [ -f "$saved" ]; then
    git apply "$saved"
    rm "$saved"
  fi
