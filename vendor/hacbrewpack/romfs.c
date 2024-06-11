#include <string.h>
#include <stdlib.h>
#include <inttypes.h>
#include <stdio.h>
#include <dirent.h>
#include "types.h"
#include "romfs.h"
#include "utils.h"
#include <sys/stat.h>

#define ROMFS_ENTRY_EMPTY 0xFFFFFFFF
#define ROMFS_FILEPARTITION_OFS 0x200

romfs_direntry_t *romfs_get_direntry(romfs_direntry_t *directories, uint32_t offset)
{
    return (romfs_direntry_t *)((char *)directories + offset);
}

romfs_fentry_t *romfs_get_fentry(romfs_fentry_t *files, uint32_t offset)
{
    return (romfs_fentry_t *)((char *)files + offset);
}

uint32_t calc_path_hash(uint32_t parent, const unsigned char *path, uint32_t start, size_t path_len)
{
    uint32_t hash = parent ^ 123456789;
    for (uint32_t i = 0; i < path_len; i++)
    {
        hash = (hash >> 5) | (hash << 27);
        hash ^= path[start + i];
    }

    return hash;
}

uint32_t align(uint32_t offset, uint32_t alignment)
{
    uint32_t mask = ~(alignment - 1);

    return (offset + (alignment - 1)) & mask;
}

uint64_t align64(uint64_t offset, uint64_t alignment)
{
    uint64_t mask = ~(uint64_t)(alignment - 1);

    return (offset + (alignment - 1)) & mask;
}

uint32_t romfs_get_hash_table_count(uint32_t num_entries)
{
    if (num_entries < 3)
    {
        return 3;
    }
    else if (num_entries < 19)
    {
        return num_entries | 1;
    }
    uint32_t count = num_entries;
    while (count % 2 == 0 || count % 3 == 0 || count % 5 == 0 || count % 7 == 0 || count % 11 == 0 || count % 13 == 0 || count % 17 == 0)
    {
        count++;
    }
    return count;
}

void romfs_visit_dir(romfs_dirent_ctx_t *parent, romfs_ctx_t *romfs_ctx)
{
    osdir_t *dir = NULL;
    osdirent_t *cur_dirent = NULL;
    romfs_dirent_ctx_t *child_dir_tree = NULL;
    romfs_fent_ctx_t *child_file_tree = NULL;
    romfs_dirent_ctx_t *cur_dir = NULL;
    romfs_fent_ctx_t *cur_file = NULL;
    filepath_t cur_path;
    filepath_t cur_sum_path;

    os_stat64_t cur_stats;

    if ((dir = os_opendir(parent->sum_path.os_path)) == NULL)
    {
        fprintf(stderr, "Failed to open directory %s!\n", parent->sum_path.char_path);
        exit(EXIT_FAILURE);
    }

    while ((cur_dirent = os_readdir(dir)))
    {
        filepath_init(&cur_path);
        filepath_set(&cur_path, "");
        filepath_os_append(&cur_path, cur_dirent->d_name);

        if (strcmp(cur_path.char_path, "/.") == 0 || strcmp(cur_path.char_path, "/..") == 0 || strcmp(cur_path.char_path, "\\.") == 0 || strcmp(cur_path.char_path, "\\..") == 0)
        {
            /* Special case . and .. */
            continue;
        }

        filepath_copy(&cur_sum_path, &parent->sum_path);
        filepath_os_append(&cur_sum_path, cur_dirent->d_name);

        if (os_stat(cur_sum_path.os_path, &cur_stats) == -1)
        {
            fprintf(stderr, "Failed to stat %s\n", cur_sum_path.char_path);
            exit(EXIT_FAILURE);
        }

        if ((cur_stats.st_mode & S_IFMT) == S_IFDIR)
        {
            /* Directory */
            if ((cur_dir = calloc(1, sizeof(romfs_dirent_ctx_t))) == NULL)
            {
                fprintf(stderr, "Failed to allocate RomFS directory context!\n");
                exit(EXIT_FAILURE);
            }

            romfs_ctx->num_dirs++;

            cur_dir->parent = parent;
            filepath_copy(&cur_dir->sum_path, &cur_sum_path);
            filepath_copy(&cur_dir->cur_path, &cur_path);

            romfs_ctx->dir_table_size += 0x18 + align(strlen(cur_dir->cur_path.char_path) - 1, 4);

            /* Ordered insertion on sibling */
            if (child_dir_tree == NULL || strcmp(cur_dir->sum_path.char_path, child_dir_tree->sum_path.char_path) < 0)
            {
                cur_dir->sibling = child_dir_tree;
                child_dir_tree = cur_dir;
            }
            else
            {
                romfs_dirent_ctx_t *child, *prev;
                prev = child_dir_tree;
                child = child_dir_tree->sibling;
                while (child != NULL)
                {
                    if (strcmp(cur_dir->sum_path.char_path, child->sum_path.char_path) < 0)
                    {
                        break;
                    }
                    prev = child;
                    child = child->sibling;
                }

                prev->sibling = cur_dir;
                cur_dir->sibling = child;
            }

            /* Ordered insertion on next */
            romfs_dirent_ctx_t *tmp = parent->next, *tmp_prev = parent;
            while (tmp != NULL)
            {
                if (strcmp(cur_dir->sum_path.char_path, tmp->sum_path.char_path) < 0)
                {
                    break;
                }
                tmp_prev = tmp;
                tmp = tmp->next;
            }
            tmp_prev->next = cur_dir;
            cur_dir->next = tmp;

            cur_dir = NULL;
        }
        else if ((cur_stats.st_mode & S_IFMT) == S_IFREG)
        {
            /* File */
            if ((cur_file = calloc(1, sizeof(romfs_fent_ctx_t))) == NULL)
            {
                fprintf(stderr, "Failed to allocate RomFS File context!\n");
                exit(EXIT_FAILURE);
            }

            romfs_ctx->num_files++;

            cur_file->parent = parent;
            filepath_copy(&cur_file->sum_path, &cur_sum_path);
            filepath_copy(&cur_file->cur_path, &cur_path);
            cur_file->size = cur_stats.st_size;

            romfs_ctx->file_table_size += 0x20 + align(strlen(cur_file->cur_path.char_path) - 1, 4);

            /* Ordered insertion on sibling */
            if (child_file_tree == NULL || strcmp(cur_file->sum_path.char_path, child_file_tree->sum_path.char_path) < 0)
            {
                cur_file->sibling = child_file_tree;
                child_file_tree = cur_file;
            }
            else
            {
                romfs_fent_ctx_t *child, *prev;
                prev = child_file_tree;
                child = child_file_tree->sibling;
                while (child != NULL)
                {
                    if (strcmp(cur_file->sum_path.char_path, child->sum_path.char_path) < 0)
                    {
                        break;
                    }
                    prev = child;
                    child = child->sibling;
                }

                prev->sibling = cur_file;
                cur_file->sibling = child;
            }

            /* Ordered insertion on next */
            if (romfs_ctx->files == NULL || strcmp(cur_file->sum_path.char_path, romfs_ctx->files->sum_path.char_path) < 0)
            {
                cur_file->next = romfs_ctx->files;
                romfs_ctx->files = cur_file;
            }
            else
            {
                romfs_fent_ctx_t *child, *prev;
                prev = romfs_ctx->files;
                child = romfs_ctx->files->next;
                while (child != NULL)
                {
                    if (strcmp(cur_file->sum_path.char_path, child->sum_path.char_path) < 0)
                    {
                        break;
                    }
                    prev = child;
                    child = child->next;
                }

                prev->next = cur_file;
                cur_file->next = child;
            }

            cur_file = NULL;
        }
        else
        {
            fprintf(stderr, "Invalid FS object type for %s!\n", cur_path.char_path);
            exit(EXIT_FAILURE);
        }
    }

    os_closedir(dir);
    parent->child = child_dir_tree;
    parent->file = child_file_tree;

    cur_dir = child_dir_tree;
    while (cur_dir != NULL)
    {
        romfs_visit_dir(cur_dir, romfs_ctx);
        cur_dir = cur_dir->sibling;
    }
}

size_t build_romfs_into_file(filepath_t *in_dirpath, FILE *f_out, off_t base_offset, filepath_t *out_romfspath)
{
    romfs_dirent_ctx_t *root_ctx = calloc(1, sizeof(romfs_dirent_ctx_t));
    if (root_ctx == NULL)
    {
        fprintf(stderr, "Failed to allocate root context!\n");
        exit(EXIT_FAILURE);
    }

    root_ctx->parent = root_ctx;

    romfs_ctx_t romfs_ctx;
    memset(&romfs_ctx, 0, sizeof(romfs_ctx));

    filepath_copy(&root_ctx->sum_path, in_dirpath);
    filepath_init(&root_ctx->cur_path);
    filepath_set(&root_ctx->cur_path, "");
    romfs_ctx.dir_table_size = 0x18; /* Root directory. */
    romfs_ctx.num_dirs = 1;

    /* Visit all directories. */
    printf("Visiting directories\n");
    romfs_visit_dir(root_ctx, &romfs_ctx);
    uint32_t dir_hash_table_entry_count = romfs_get_hash_table_count(romfs_ctx.num_dirs);
    uint32_t file_hash_table_entry_count = romfs_get_hash_table_count(romfs_ctx.num_files);
    romfs_ctx.dir_hash_table_size = 4 * dir_hash_table_entry_count;
    romfs_ctx.file_hash_table_size = 4 * file_hash_table_entry_count;

    romfs_header_t header;
    memset(&header, 0, sizeof(header));
    romfs_fent_ctx_t *cur_file = NULL;
    romfs_dirent_ctx_t *cur_dir = NULL;
    uint32_t entry_offset = 0;

    uint32_t *dir_hash_table = malloc(romfs_ctx.dir_hash_table_size);
    if (dir_hash_table == NULL)
    {
        fprintf(stderr, "Failed to allocate directory hash table!\n");
        exit(EXIT_FAILURE);
    }

    for (uint32_t i = 0; i < dir_hash_table_entry_count; i++)
    {
        dir_hash_table[i] = le_word(ROMFS_ENTRY_EMPTY);
    }

    uint32_t *file_hash_table = malloc(romfs_ctx.file_hash_table_size);
    if (file_hash_table == NULL)
    {
        fprintf(stderr, "Failed to allocate file hash table!\n");
        exit(EXIT_FAILURE);
    }

    for (uint32_t i = 0; i < file_hash_table_entry_count; i++)
    {
        file_hash_table[i] = le_word(ROMFS_ENTRY_EMPTY);
    }

    romfs_direntry_t *dir_table = calloc(1, romfs_ctx.dir_table_size);
    if (dir_table == NULL)
    {
        fprintf(stderr, "Failed to allocate directory table!\n");
        exit(EXIT_FAILURE);
    }

    romfs_fentry_t *file_table = calloc(1, romfs_ctx.file_table_size);
    if (file_table == NULL)
    {
        fprintf(stderr, "Failed to allocate file table!\n");
        exit(EXIT_FAILURE);
    }

    printf("Calculating metadata\n");
    /* Determine file offsets. */
    cur_file = romfs_ctx.files;
    entry_offset = 0;
    while (cur_file != NULL)
    {
        romfs_ctx.file_partition_size = align64(romfs_ctx.file_partition_size, 0x10);
        cur_file->offset = romfs_ctx.file_partition_size;
        romfs_ctx.file_partition_size += cur_file->size;
        cur_file->entry_offset = entry_offset;
        entry_offset += 0x20 + align(strlen(cur_file->cur_path.char_path) - 1, 4);
        cur_file = cur_file->next;
    }

    /* Determine dir offsets. */
    cur_dir = root_ctx;
    entry_offset = 0;
    while (cur_dir != NULL)
    {
        cur_dir->entry_offset = entry_offset;
        if (cur_dir == root_ctx)
        {
            entry_offset += 0x18;
        }
        else
        {
            entry_offset += 0x18 + align(strlen(cur_dir->cur_path.char_path) - 1, 4);
        }
        cur_dir = cur_dir->next;
    }

    /* Populate file tables. */
    cur_file = romfs_ctx.files;
    while (cur_file != NULL)
    {
        romfs_fentry_t *cur_entry = romfs_get_fentry(file_table, cur_file->entry_offset);
        cur_entry->parent = le_word(cur_file->parent->entry_offset);
        cur_entry->sibling = le_word(cur_file->sibling == NULL ? ROMFS_ENTRY_EMPTY : cur_file->sibling->entry_offset);
        cur_entry->offset = le_dword(cur_file->offset);
        cur_entry->size = le_dword(cur_file->size);

        uint32_t name_size = strlen(cur_file->cur_path.char_path) - 1;
        uint32_t hash = calc_path_hash(cur_file->parent->entry_offset, (unsigned char *)cur_file->cur_path.char_path, 1, name_size);
        cur_entry->hash = file_hash_table[hash % file_hash_table_entry_count];
        file_hash_table[hash % file_hash_table_entry_count] = le_word(cur_file->entry_offset);

        cur_entry->name_size = name_size;
        memcpy(cur_entry->name, cur_file->cur_path.char_path + 1, name_size);

        cur_file = cur_file->next;
    }

    /* Populate dir tables. */
    cur_dir = root_ctx;
    while (cur_dir != NULL)
    {
        romfs_direntry_t *cur_entry = romfs_get_direntry(dir_table, cur_dir->entry_offset);
        cur_entry->parent = le_word(cur_dir->parent->entry_offset);
        cur_entry->sibling = le_word(cur_dir->sibling == NULL ? ROMFS_ENTRY_EMPTY : cur_dir->sibling->entry_offset);
        cur_entry->child = le_word(cur_dir->child == NULL ? ROMFS_ENTRY_EMPTY : cur_dir->child->entry_offset);
        cur_entry->file = le_word(cur_dir->file == NULL ? ROMFS_ENTRY_EMPTY : cur_dir->file->entry_offset);

        uint32_t name_size = (cur_dir == root_ctx) ? 0 : strlen(cur_dir->cur_path.char_path) - 1;
        uint32_t hash = calc_path_hash((cur_dir == root_ctx) ? 0 : cur_dir->parent->entry_offset, (unsigned char *)cur_dir->cur_path.char_path, 1, name_size);
        cur_entry->hash = dir_hash_table[hash % dir_hash_table_entry_count];
        dir_hash_table[hash % dir_hash_table_entry_count] = le_word(cur_dir->entry_offset);

        cur_entry->name_size = name_size;
        memcpy(cur_entry->name, cur_dir->cur_path.char_path + 1, name_size);

        romfs_dirent_ctx_t *temp = cur_dir;
        cur_dir = cur_dir->next;
        free(temp);
    }

    header.header_size = le_dword(sizeof(header));
    header.file_hash_table_size = le_dword(romfs_ctx.file_hash_table_size);
    header.file_table_size = le_dword(romfs_ctx.file_table_size);
    header.dir_hash_table_size = le_dword(romfs_ctx.dir_hash_table_size);
    header.dir_table_size = le_dword(romfs_ctx.dir_table_size);
    header.file_partition_ofs = le_dword(ROMFS_FILEPARTITION_OFS);

    /* Abuse of endianness follows. */
    uint64_t dir_hash_table_ofs = align64(romfs_ctx.file_partition_size + ROMFS_FILEPARTITION_OFS, 4);
    header.dir_hash_table_ofs = dir_hash_table_ofs;
    header.dir_table_ofs = header.dir_hash_table_ofs + romfs_ctx.dir_hash_table_size;
    header.file_hash_table_ofs = header.dir_table_ofs + romfs_ctx.dir_table_size;
    header.file_table_ofs = header.file_hash_table_ofs + romfs_ctx.file_hash_table_size;
    header.dir_hash_table_ofs = le_dword(header.dir_hash_table_ofs);
    header.dir_table_ofs = le_dword(header.dir_table_ofs);
    header.file_hash_table_ofs = le_dword(header.file_hash_table_ofs);
    header.file_table_ofs = le_dword(header.file_table_ofs);

    fseeko64(f_out, base_offset, SEEK_SET);
    fwrite(&header, 1, sizeof(header), f_out);

    /* Write files. */
    uint64_t read_size = 0x61A8000;            // 100 MB
    unsigned char *buffer = malloc(read_size); // 100 MB buffer.
    if (buffer == NULL)
    {
        fprintf(stderr, "Failed to allocate work buffer!\n");
        exit(EXIT_FAILURE);
    }
    cur_file = romfs_ctx.files;
    while (cur_file != NULL)
    {
        FILE *f_in = os_fopen(cur_file->sum_path.os_path, OS_MODE_READ);
        if (f_in == NULL)
        {
            fprintf(stderr, "Failed to open %s!\n", cur_file->sum_path.char_path);
            exit(EXIT_FAILURE);
        }

        printf("Writing %s to %s\n", cur_file->sum_path.char_path, out_romfspath->char_path);
        fseeko64(f_out, base_offset + cur_file->offset + ROMFS_FILEPARTITION_OFS, SEEK_SET);
        uint64_t offset = 0;
        while (offset < cur_file->size)
        {
            if (cur_file->size - offset < read_size)
            {
                read_size = cur_file->size - offset;
            }

            if (fread(buffer, 1, read_size, f_in) != read_size)
            {
                fprintf(stderr, "Failed to read from %s!\n", cur_file->sum_path.char_path);
                exit(EXIT_FAILURE);
            }

            if (fwrite(buffer, 1, read_size, f_out) != read_size)
            {
                fprintf(stderr, "Failed to write to output!\n");
                exit(EXIT_FAILURE);
            }

            offset += read_size;
        }

        os_fclose(f_in);

        romfs_fent_ctx_t *temp = cur_file;
        cur_file = cur_file->next;
        free(temp);
    }
    free(buffer);

    fseeko64(f_out, base_offset + dir_hash_table_ofs, SEEK_SET);
    if (fwrite(dir_hash_table, 1, romfs_ctx.dir_hash_table_size, f_out) != romfs_ctx.dir_hash_table_size)
    {
        fprintf(stderr, "Failed to write dir hash table!\n");
        exit(EXIT_FAILURE);
    }
    free(dir_hash_table);

    if (fwrite(dir_table, 1, romfs_ctx.dir_table_size, f_out) != romfs_ctx.dir_table_size)
    {
        fprintf(stderr, "Failed to write dir table!\n");
        exit(EXIT_FAILURE);
    }
    free(dir_table);

    if (fwrite(file_hash_table, 1, romfs_ctx.file_hash_table_size, f_out) != romfs_ctx.file_hash_table_size)
    {
        fprintf(stderr, "Failed to write file hash table!\n");
        exit(EXIT_FAILURE);
    }
    free(file_hash_table);

    if (fwrite(file_table, 1, romfs_ctx.file_table_size, f_out) != romfs_ctx.file_table_size)
    {
        fprintf(stderr, "Failed to write file table!\n");
        exit(EXIT_FAILURE);
    }
    free(file_table);

    return dir_hash_table_ofs + romfs_ctx.dir_hash_table_size + romfs_ctx.dir_table_size + romfs_ctx.file_hash_table_size + romfs_ctx.file_table_size;
}

size_t romfs_build(filepath_t *in_dirpath, filepath_t *out_romfspath, uint64_t *out_size)
{
    FILE *f_out = NULL;

    if ((f_out = os_fopen(out_romfspath->os_path, OS_MODE_WRITE)) == NULL)
    {
        fprintf(stderr, "Failed to open %s!\n", out_romfspath->char_path);
        exit(EXIT_FAILURE);
    }

    size_t sz = build_romfs_into_file(in_dirpath, f_out, 0, out_romfspath);

    // Write Padding
    fseeko64(f_out, 0, SEEK_END);
    *out_size = (uint64_t)ftello64(f_out);
    uint64_t hash_block_size = IVFC_HASH_BLOCK_SIZE;
    uint64_t curr_offset = ftello64(f_out);
    uint64_t padding_size = hash_block_size - (curr_offset % hash_block_size);
    if (padding_size != 0)
    {
        unsigned char *padding_buf = (unsigned char *)calloc(1, padding_size);
        fwrite(padding_buf, 1, padding_size, f_out);
        free(padding_buf);
    }

    fclose(f_out);
    return sz;
}