_default:
  just -l

# builds all vendor dependencies
vendor:
  for c in $(just --summary | xargs -n1 | grep vendor-); do just $c; done

vendor-nro:
  cd vendor/Forwarder-Mod && (make clean; make all)
vendor-hacbrewpack:
  cd vendor/hacbrewpack && (make clean_full; make)

fetch-titles:
  npm exec tsx vendor/tinfoil/update.ts
