_default:
  just -l

@_check CMD MSG="":
    if ! command -v {{CMD}} >/dev/null 2>&1 /dev/null; then echo "{{CMD}} is required! {{MSG}}"; exit 1; fi

# set up the local repository for development
setup:
  npm install --force
  echo "#!/usr/bin/env bash" > .git/hooks/pre-commit
  echo "just pre-commit" >> .git/hooks/pre-commit
  chmod +x .git/hooks/pre-commit

  @just _check "gcc"
  @just _check "make"
  @just _check "emcc" "See https://emscripten.org/docs/getting_started/downloads.html for installation instructions"
  @just _check "dkp-pacman" "See https://devkitpro.org/wiki/Getting_Started for installation instructions"

#
# Dev Scripts
#

# start the app in dev mode
dev *args:
  npm start -- -- {{args}}

# runs all tests and checks
test:
  npm run test

# runs vitest in watch mode
testw:
  npm run test:vitest

# runs vitest in bench mode
bench:
  npm exec vitest -- bench

bench100m:
  npm exec tsx -- ./scripts/bench100m.ts

# formats all code
format:
  npm run format

# creates a fake NAND dump for testing
create-nand *ARGS:
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
  cd src/nand/xtsn && npm run prepublish && npm publish

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
  just test
  if [ -f "$saved" ]; then
    git apply "$saved"
    rm "$saved"
  fi
