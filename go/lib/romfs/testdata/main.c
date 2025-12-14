#include <stdio.h>
#include <stdlib.h>
#include "romfs.h"
#include "filepath.h"

size_t build_romfs_into_file(filepath_t *in_dirpath, FILE *f_out, off_t base_offset, filepath_t *out_romfspath);

int main(int argc, char **argv) {
    if (argc < 3) {
        fprintf(stderr, "Usage: %s <input_dir> <output_file>\n", argv[0]);
        return 1;
    }

    filepath_t in_dirpath;
    filepath_init(&in_dirpath);
    filepath_set(&in_dirpath, argv[1]);

    filepath_t out_romfspath;
    filepath_init(&out_romfspath);
    filepath_set(&out_romfspath, argv[2]);

    FILE *f_out = fopen(out_romfspath.char_path, "wb");
    if (f_out == NULL) {
        perror("Failed to open output file");
        return 1;
    }

    build_romfs_into_file(&in_dirpath, f_out, 0, &out_romfspath);

    fclose(f_out);
    return 0;
}
