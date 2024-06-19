_default:
  just -l

# TODO: check for DKP
# TODO: check for build toolchain
# TODO: check for emscripten for wasm
# set up the local repository for development
setup:
  npm install --force
  echo "#!/usr/bin/env bash" > .git/hooks/pre-commit
  echo "just pre-commit" >> .git/hooks/pre-commit
  chmod +x .git/hooks/pre-commit

#
# Dev Scripts
#

# runs all tests and checks
test:
  npm run test

# runs vitest in watch mode
testw:
  npm run test:vitest

# formats all code
format:
  npm run format

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
