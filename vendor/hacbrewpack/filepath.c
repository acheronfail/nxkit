#include <sys/stat.h>
#include <stdio.h>
#include <string.h>
#include <stdarg.h>
#include "types.h"
#include "filepath.h"
#include "ConvertUTF.h"
#include <unistd.h>

void os_strcpy(oschar_t *dst, const char *src)
{
#ifdef _WIN32
    if (src == NULL)
        return;

    const UTF8 *sourceStart = (const UTF8 *)src;
    UTF16 *targetStart = (UTF16 *)dst;
    uint32_t src_len, dst_len;
    src_len = strlen(src);
    dst_len = src_len + 1;
    const UTF8 *sourceEnd = (const UTF8 *)(src + src_len);
    UTF16 *targetEnd = (UTF16 *)(dst + dst_len);

    if (ConvertUTF8toUTF16(&sourceStart, sourceEnd, &targetStart, targetEnd, 0) != conversionOK)
    {
        fprintf(stderr, "Failed to convert %s to UTF-16!\n", src);
        exit(EXIT_FAILURE);
    }
#else
    strcpy(dst, src);
#endif
}

void os_strncpy_to_char(char *dst, const oschar_t *src, size_t size)
{
#ifdef _WIN32
    WideCharToMultiByte(CP_UTF8, 0, src, -1, dst, size, NULL, NULL);
#else
    strncpy(dst, src, size);
#endif
}

int os_makedir(const oschar_t *dir)
{
#ifdef _WIN32
    return _wmkdir(dir);
#else
    return mkdir(dir, 0777);
#endif
}

int os_rmdir(const oschar_t *dir)
{
#ifdef _WIN32
    return _wrmdir(dir);
#else
    return remove(dir);
#endif
}

void filepath_update(filepath_t *fpath)
{
    memset(fpath->os_path, 0, MAX_PATH * sizeof(oschar_t));
    os_strcpy(fpath->os_path, fpath->char_path);
}

void filepath_init(filepath_t *fpath)
{
    fpath->valid = VALIDITY_INVALID;
}

void filepath_copy(filepath_t *fpath, filepath_t *copy)
{
    if (copy != NULL && copy->valid == VALIDITY_VALID)
        memcpy(fpath, copy, sizeof(filepath_t));
    else
        memset(fpath, 0, sizeof(filepath_t));
}

void filepath_append(filepath_t *fpath, const char *format, ...)
{
    char tmppath[MAX_PATH];
    va_list args;

    if (fpath->valid == VALIDITY_INVALID)
        return;

    memset(tmppath, 0, MAX_PATH);

    va_start(args, format);
    vsnprintf(tmppath, sizeof(tmppath), format, args);
    va_end(args);

    strcat(fpath->char_path, OS_PATH_SEPARATOR);
    strcat(fpath->char_path, tmppath);
    filepath_update(fpath);
}

void filepath_append_n(filepath_t *fpath, uint32_t n, const char *format, ...)
{
    char tmppath[MAX_PATH];
    va_list args;

    if (fpath->valid == VALIDITY_INVALID || n > MAX_PATH)
        return;

    memset(tmppath, 0, MAX_PATH);

    va_start(args, format);
    vsnprintf(tmppath, sizeof(tmppath), format, args);
    va_end(args);

    strcat(fpath->char_path, OS_PATH_SEPARATOR);
    strncat(fpath->char_path, tmppath, n);
    filepath_update(fpath);
}

void filepath_set(filepath_t *fpath, const char *path)
{
    if (strlen(path) < MAX_PATH)
    {
        fpath->valid = VALIDITY_VALID;
        memset(fpath->char_path, 0, MAX_PATH);
        strncpy(fpath->char_path, path, MAX_PATH - 1);
        filepath_update(fpath);
    }
    else
    {
        fpath->valid = VALIDITY_INVALID;
    }
}

oschar_t *filepath_get(filepath_t *fpath)
{
    if (fpath->valid == VALIDITY_INVALID)
        return NULL;
    else
        return fpath->os_path;
}

void filepath_os_append(filepath_t *fpath, oschar_t *path)
{
    char tmppath[MAX_SWITCHPATH];
    if (fpath->valid == VALIDITY_INVALID)
        return;

    memset(tmppath, 0, MAX_SWITCHPATH);

    os_strncpy_to_char(tmppath, path, MAX_SWITCHPATH);
    strcat(fpath->char_path, OS_PATH_SEPARATOR);
    strcat(fpath->char_path, tmppath);
    filepath_update(fpath);
}

// Original Code by asveikau https://stackoverflow.com/users/182748/asveikau
int filepath_remove_directory(filepath_t *dir_path)
{
    filepath_t dir_path_cpy;
    filepath_init(&dir_path_cpy);
    filepath_copy(&dir_path_cpy, dir_path);
    if (strcmp(&dir_path_cpy.char_path[strlen(dir_path_cpy.char_path) - 1], OS_PATH_SEPARATOR) != 0)
        filepath_append(&dir_path_cpy, "");
    DIR *d = opendir(dir_path_cpy.char_path);
    size_t path_len = strlen(dir_path_cpy.char_path);
    int r = -1;

    if (d)
    {
        struct dirent *p;

        r = 0;

        while (!r && (p = readdir(d)))
        {
            int r2 = -1;
            char *buf;
            size_t len;

            /* Skip the names "." and ".." as we don't want to recurse on them. */
            if (!strcmp(p->d_name, ".") || !strcmp(p->d_name, ".."))
            {
                continue;
            }

            len = path_len + strlen(p->d_name) + 2;
            buf = malloc(len);

            if (buf)
            {
                struct stat statbuf;

                snprintf(buf, len, "%s%s", dir_path_cpy.char_path, p->d_name);

                if (!stat(buf, &statbuf))
                {
                    if (S_ISDIR(statbuf.st_mode))
                    {
                        filepath_t buf_dir;
                        filepath_init(&buf_dir);
                        filepath_set(&buf_dir, buf);
                        r2 = filepath_remove_directory(&buf_dir);
                    }
                    else
                    {
                        r2 = os_deletefile(buf);
                    }
                }

                free(buf);
            }

            r = r2;
        }

        closedir(d);
    }

    if (!r)
    {
        r = os_rmdir(dir_path_cpy.os_path);
    }

    return r;
}

void filepath_copy_file(filepath_t *source_file, filepath_t *destination_path)
{
    uint64_t file_size;
    FILE *source;
    FILE *dst;
    source = os_fopen(source_file->os_path, OS_MODE_READ);
    dst = os_fopen(destination_path->os_path, OS_MODE_WRITE);

    if (source == NULL)
    {
        fprintf(stderr, "Failed to open %s!\n", source_file->char_path);
        exit(EXIT_FAILURE);
    }
    if (dst == NULL)
    {
        fprintf(stderr, "Failed to open %s!\n", destination_path->char_path);
        exit(EXIT_FAILURE);
    }

    fseeko64(source, 0, SEEK_END);
    file_size = ftello64(source);
    fseeko64(source, 0, SEEK_SET);

    uint64_t read_size = 0x61A8000; // 100 MB buffer.
    unsigned char *buf = malloc(read_size);
    if (buf == NULL)
    {
        fprintf(stderr, "Failed to allocate file-read buffer!\n");
        exit(EXIT_FAILURE);
    }

    uint64_t ofs = 0;
    while (ofs < file_size)
    {
        if (ofs + read_size >= file_size)
            read_size = file_size - ofs;
        if (fread(buf, 1, read_size, source) != read_size)
        {
            fprintf(stderr, "Failed to read file %s\n", source_file->char_path);
            exit(EXIT_FAILURE);
        }
        fwrite(buf, read_size, 1, dst);
        ofs += read_size;
    }

    free(buf);
    fclose(source);
    fclose(dst);
}