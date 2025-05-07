#!/bin/sh

set -euo pipefail

fixtures_dir="/tmp/fixtures"
mkdir -p "${fixtures_dir}"

echo 'text file' > "${fixtures_dir}/INFO.TXT"
dd if=/dev/zero of="${fixtures_dir}/a file with a long name.dat" bs=1024 count=2 2>/dev/null
dd if=/dev/zero of="${fixtures_dir}/another file" bs=1024 count=6                2>/dev/null
dd if=/dev/zero of="${fixtures_dir}/a file with a long name.dat" bs=1024 count=7 2>/dev/null
dd if=/dev/zero of="${fixtures_dir}/some_long_embedded_nameא" bs=1024 count=7    2>/dev/null

for n in $(echo "12 16 32"); do
  out="/mnt/fat${n}"
  rm -rf "${out}"

  disk="${out}/disk.img"

  mkdir -p ${out}
  dd if=/dev/zero of="${disk}" bs=1M count=$(( n + 2 ))                          2>/dev/null
  mkfs.vfat -v -F ${n} "${disk}" >/dev/null

  mmd -i "${disk}" ::/dir
  mmd -i "${disk}" ::/dir/subdir
  mcopy -i "${disk}" "${fixtures_dir}/INFO.TXT" ::/
  mcopy -i "${disk}" "${fixtures_dir}/a file with a long name.dat" ::/
  mcopy -i "${disk}" "${fixtures_dir}/another file" ::/
  mcopy -i "${disk}" "${fixtures_dir}/some_long_embedded_nameא" ::/dir/subdir

  mmd   -i "${disk}" ::/lower83
  mcopy -i "${disk}" "${fixtures_dir}/INFO.TXT" ::/lower83/lower.low
  mcopy -i "${disk}" "${fixtures_dir}/INFO.TXT" ::/lower83/lower.UPP
  mcopy -i "${disk}" "${fixtures_dir}/INFO.TXT" ::/lower83/UPPER.low
  mcopy -i "${disk}" "${fixtures_dir}/INFO.TXT" ::/lower83/UPPER.UPP

  mmd   -i "${disk}" ::/mkdir

  i=0
  until [ $i -gt 75 ]; do mmd -i "${disk}" ::/dir/subdir_${i}; i=$(( $i+1 )); done

  fatlabel "${disk}" "FAT${n}-TEST"
  fsstat -f fat${n} "${disk}" > "${out}/fsstat.txt"
done
