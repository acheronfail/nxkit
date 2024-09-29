nxkit_image := 'nxkit'
node_arch := if arch() == 'x86_64' { 'x64' } else { 'arm64' }

_default:
  just -l

@_check CMD MSG="":
    if ! command -v {{CMD}} >/dev/null 2>&1 /dev/null; then echo "{{CMD}} is required! {{MSG}}"; exit 1; fi

# set up the local repository for development
setup:
  npm install --force
  if [ -z "${CI:-}" ]; then just setup_hooks; fi

# set up git hooks
setup_hooks:
  @echo "setting up git hooks..."
  @echo "#!/usr/bin/env bash" > .git/hooks/pre-commit
  @echo "just pre-commit" >> .git/hooks/pre-commit
  @chmod +x .git/hooks/pre-commit
  @env echo -n "Do you want to create a fake nand dump? (y/N): "; read ans; if [[ $ans = *y* ]]; then just create-nand; fi

# clean up installed toolchains and installed dependencies
clean:
  rm -rf node_modules

# revert to the state after cloning the repository
clean_all: clean
  git clean -fdx

#
# Dev Scripts
#

# run npm
npm *args:
  npm {{args}}

# start the app in dev mode
dev *args: rebuild-electron
  npm start -- -- {{args}}

# package the app and start it
dev-packaged *args:
  rm -rf out
  npm run package
  ./out/NXKit-{{os()}}-{{node_arch}}/nxkit {{args}}

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
test *ARGS: rebuild-node
  npm run test:vitest -- {{ARGS}}

# runs benchmarks; outputs a .cpuprofile file and creates bench.json if title was passed
bench TITLE='': rebuild-node
  @mkdir -p scripts/build/Release
  @cp src/node/nand/xtsn/build/Release/xtsn.node scripts/build/Release/xtsn.node
  @cp node_modules/js-fatfs/dist/fatfs.wasm scripts/
  npx esbuild --bundle --platform=node --format=esm scripts/bench100m.ts --outfile=scripts/bench100m.js
  node --cpu-prof scripts/bench100m.js {{TITLE}}

# formats all code
format:
  npm run format

# creates a fake NAND dump for testing
create-nand *ARGS: rebuild-node
  npm exec tsx scripts/create-fake-nand.ts -- {{ARGS}}

#
# Vendor
#

_nxkit_image:
  docker build --platform linux/amd64 --tag {{nxkit_image}} vendor/docker

# rebuilds all vendor dependencies
vendor: _nxkit_image
  for c in $(just --summary | xargs -n1 | grep vendor-); do just $c; done

vendor-nro: _nxkit_image
  docker run -ti --rm -v "$PWD/vendor/Forwarder-Mod:/src" {{nxkit_image}} bash -c '(cd /src; make clean; make all)'
vendor-hacbrewpack: _nxkit_image
  docker run -ti --rm -v "$PWD/vendor/hacbrewpack:/src" {{nxkit_image}} bash -c '(cd /src; make clean_full; make)'

fetch-titles:
  npm exec tsx vendor/tinfoil/update.ts

#
# Release
#

publish-xtsn:
  cd src/node/nand/xtsn && npm run prepublish && npm publish

package:
  npm run make

publish:
  npm run publish

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
