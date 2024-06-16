_default:
  just -l

# builds vendor dependencies
build:
  cd vendor/hacbrewpack && (make clean_full; make)
