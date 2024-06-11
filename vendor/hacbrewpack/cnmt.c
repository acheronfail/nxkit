#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include "cnmt.h"
#include "filepath.h"
#include "utils.h"

void cnmt_create(cnmt_ctx_t *cnmt_ctx, filepath_t *cnmt_filepath, hbp_settings_t *settings)
{
    cnmt_extended_application_header_t cnmt_ext_header;
    memset(&cnmt_ext_header, 0, sizeof(cnmt_ext_header));

    // Common values
    cnmt_ctx->cnmt_header.type = 0x80; // Application
    cnmt_ctx->cnmt_header.extended_header_size = 0x10;
    cnmt_ctx->cnmt_header.content_entry_count = 0x2;
    if (settings->htmldoc_romfs_dir.valid == VALIDITY_VALID)
        cnmt_ctx->cnmt_header.content_entry_count += 0x1;
    if (settings->legalinfo_romfs_dir.valid == VALIDITY_VALID)
        cnmt_ctx->cnmt_header.content_entry_count += 0x1;
    cnmt_ext_header.patch_title_id = cnmt_ctx->cnmt_header.title_id + 0x800;

    // Set content record types
    cnmt_ctx->cnmt_content_records[0].type = 0x1;   // Program
    cnmt_ctx->cnmt_content_records[1].type = 0x3;   // Control
    cnmt_ctx->cnmt_content_records[3].type = 0x4;   // HtmlDocument
    cnmt_ctx->cnmt_content_records[4].type = 0x5;   // LegalInfo 

    printf("Writing metadata header\n");
    FILE *cnmt_file;
    cnmt_file = os_fopen(cnmt_filepath->os_path, OS_MODE_WRITE);

    if (cnmt_file != NULL)
    {
        fwrite(&cnmt_ctx->cnmt_header, 1, sizeof(cnmt_header_t), cnmt_file);
        fwrite(&cnmt_ext_header, 1, sizeof(cnmt_extended_application_header_t), cnmt_file);
    }
    else
    {
        fprintf(stderr, "Failed to create %s!\n", cnmt_filepath->char_path);
        exit(EXIT_FAILURE);
    }

    // Write content records
    uint8_t digest[0x20] = {0}; // Unknown value
    printf("Writing content records\n");
    fwrite(&cnmt_ctx->cnmt_content_records[0], sizeof(cnmt_content_record_t), 1, cnmt_file);
    fwrite(&cnmt_ctx->cnmt_content_records[1], sizeof(cnmt_content_record_t), 1, cnmt_file);
    if (settings->htmldoc_romfs_dir.valid == VALIDITY_VALID)
        fwrite(&cnmt_ctx->cnmt_content_records[3], sizeof(cnmt_content_record_t), 1, cnmt_file);
    if (settings->legalinfo_romfs_dir.valid == VALIDITY_VALID)
        fwrite(&cnmt_ctx->cnmt_content_records[4], sizeof(cnmt_content_record_t), 1, cnmt_file);    
    fwrite(digest, 1, 0x20, cnmt_file);

    fclose(cnmt_file);
}