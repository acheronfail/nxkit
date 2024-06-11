#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <inttypes.h>
#include "nca.h"
#include "sha.h"
#include "filepath.h"
#include "romfs.h"
#include "rsa.h"

void nca_create_control(hbp_settings_t *settings, cnmt_ctx_t *cnmt_ctx)
{
    printf("----> Creating Control NCA:\n");
    printf("===> Creating NCA header\n");
    nca_header_t nca_header;
    memset(&nca_header, 0, sizeof(nca_header));

    filepath_t control_nca_path;
    filepath_init(&control_nca_path);
    filepath_copy(&control_nca_path, &settings->nca_dir);
    filepath_append(&control_nca_path, "control.nca");

    FILE *control_nca_file;
    control_nca_file = os_fopen(control_nca_path.os_path, OS_MODE_WRITE_EDIT);

    // Write placeholder for NCA header
    printf("Writing NCA header placeholder to %s\n", control_nca_path.char_path);
    if (control_nca_file != NULL)
        fwrite(&nca_header, 1, sizeof(nca_header), control_nca_file);
    else
    {
        fprintf(stderr, "Failed to create %s!\n", control_nca_path.char_path);
        exit(EXIT_FAILURE);
    }

    printf("\n---> Creating Section 0:");

    // Set IVFC levels temp filepaths
    filepath_t ivfc_lvls_path[6];
    for (int a = 0; a < 6; a++)
    {
        filepath_init(&ivfc_lvls_path[a]);
        filepath_copy(&ivfc_lvls_path[a], &settings->temp_dir);
        filepath_append(&ivfc_lvls_path[a], "control_sec0_ivfc_lvl%i", a + 1);
    }

    //Build RomFS
    printf("\n===> Building RomFS\n");
    romfs_build(&settings->control_romfs_dir, &ivfc_lvls_path[5], &nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[5].hash_data_size);
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[5].block_size = 0x0E; // 0x4000

    // Create IVFC levels
    printf("\n===> Creating IVFC levels\n");
    for (int b = 4; b >= 0; b--)
    {
        printf("Writing %s\n", ivfc_lvls_path[b].char_path);
        ivfc_create_level(&ivfc_lvls_path[b], &ivfc_lvls_path[b + 1], &nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[b].hash_data_size);
        nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[b].block_size = 0x0E; // 0x4000
    }

    // Set IVFC levels logical offset
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[0].logical_offset = 0;
    for (int i = 1; i <= 5; i++)
        nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[i].logical_offset = nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[i - 1].logical_offset + nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[i - 1].hash_data_size;

    // Write IVFC levels
    printf("\n===> Writing IVFC levels\n");
    for (int c = 0; c < 6; c++)
    {
        printf("Writing %s to %s\n", ivfc_lvls_path[c].char_path, control_nca_path.char_path);
        nca_write_file(control_nca_file, &ivfc_lvls_path[c]);
    }

    // Write Padding if required
    nca_write_padding(control_nca_file);

    // Common values
    nca_header.magic = MAGIC_NCA3;
    nca_header.content_type = 0x2; // Control
    nca_header.sdk_version = settings->sdk_version;
    nca_header.title_id = cnmt_ctx->cnmt_header.title_id;
    nca_set_keygen(&nca_header, settings);

    nca_header.section_entries[0].media_start_offset = 0x6;                                          // 0xC00 / 0x200
    nca_header.section_entries[0].media_end_offset = (uint32_t)(ftello64(control_nca_file) / 0x200); // Section end offset / 200
    nca_header.section_entries[0]._0x8[0] = 0x1;                                                     // Always 1

    nca_header.fs_headers[0].hash_type = HASH_TYPE_ROMFS;
    nca_header.fs_headers[0].version = 0x2; // Always 2
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.magic = MAGIC_IVFC;
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.id = 0x20000; //Always 0x20000
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.master_hash_size = 0x20;
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.num_levels = 0x7;
    if (settings->plaintext == 0)
        nca_header.fs_headers[0].crypt_type = 0x3; // Regular crypto
    else
        nca_header.fs_headers[0].crypt_type = 0x1; // Plaintext

    // Calculate master hash and section hash
    printf("\n===> Calculating Hashes:\n");
    printf("Calculating Master hash\n");
    ivfc_calculate_master_hash(&ivfc_lvls_path[0], nca_header.fs_headers[0].romfs_superblock.ivfc_header.master_hash);
    printf("Calculating Section hash\n");
    nca_calculate_section_hash(&nca_header.fs_headers[0], nca_header.section_hashes[0]);

    printf("\n---> Finalizing:\n");

    // Set encrypted key area key 2
    memcpy(nca_header.encrypted_keys[2], settings->keyareakey, 0x10);

    printf("===> Encrypting NCA\n");
    if (settings->plaintext == 0)
    {
        // Encrypt section 0
        printf("Encrypting section 0\n");
        nca_encrypt_section(control_nca_file, &nca_header, 0);
    }

    // Encrypt header
    printf("Getting NCA file size\n");
    fseeko64(control_nca_file, 0, SEEK_END);
    nca_header.nca_size = (uint64_t)ftello64(control_nca_file);
    printf("Encrypting key area\n");
    nca_encrypt_key_area(&nca_header, settings);
    printf("Encrypting header\n");
    nca_encrypt_header(&nca_header, settings);

    // Write MCA header
    printf("\n===> Writing NCA header\n");
    printf("Writing NCA header to %s\n", control_nca_path.char_path);
    fseeko64(control_nca_file, 0, SEEK_SET);
    fwrite(&nca_header, 1, sizeof(nca_header), control_nca_file);

    // Calculate hash and nca size
    printf("\n===> Post creation process\n");
    printf("Calculating NCA hash\n");
    nca_calculate_hash(control_nca_file, cnmt_ctx, 1);

    fclose(control_nca_file);

    // Rename control.nca to ncaid.nca
    filepath_t control_nca_final_path;
    filepath_init(&control_nca_final_path);
    filepath_copy(&control_nca_final_path, &settings->nca_dir);
    char control_nca_name[37];
    hexBinaryString(cnmt_ctx->cnmt_content_records[1].ncaid, 16, control_nca_name, 33);
    strcat(control_nca_name, ".nca");
    control_nca_name[36] = '\0';
    printf("Renaming control.nca to %s\n", control_nca_name);
    filepath_append(&control_nca_final_path, "%s", control_nca_name);
    os_rename(control_nca_path.os_path, control_nca_final_path.os_path);
    printf("\n----> Created Control NCA: %s\n", control_nca_final_path.char_path);
}

void nca_create_manual_htmldoc(hbp_settings_t *settings, cnmt_ctx_t *cnmt_ctx)
{
    printf("----> Creating Manual(HtmlDoc) NCA:\n");
    printf("===> Creating NCA header\n");
    nca_header_t nca_header;
    memset(&nca_header, 0, sizeof(nca_header));

    filepath_t manual_htmldoc_nca_path;
    filepath_init(&manual_htmldoc_nca_path);
    filepath_copy(&manual_htmldoc_nca_path, &settings->nca_dir);
    filepath_append(&manual_htmldoc_nca_path, "manual_htmldoc.nca");

    FILE *manual_htmldoc_nca_file;
    manual_htmldoc_nca_file = os_fopen(manual_htmldoc_nca_path.os_path, OS_MODE_WRITE_EDIT);

    // Write placeholder for NCA header
    printf("Writing NCA header placeholder to %s\n", manual_htmldoc_nca_path.char_path);
    if (manual_htmldoc_nca_file != NULL)
        fwrite(&nca_header, 1, sizeof(nca_header), manual_htmldoc_nca_file);
    else
    {
        fprintf(stderr, "Failed to create %s!\n", manual_htmldoc_nca_path.char_path);
        exit(EXIT_FAILURE);
    }

    printf("\n---> Creating Section 0:");

    // Set IVFC levels temp filepaths
    filepath_t ivfc_lvls_path[6];
    for (int a = 0; a < 6; a++)
    {
        filepath_init(&ivfc_lvls_path[a]);
        filepath_copy(&ivfc_lvls_path[a], &settings->temp_dir);
        filepath_append(&ivfc_lvls_path[a], "manual_htmldoc_sec0_ivfc_lvl%i", a + 1);
    }

    //Build RomFS
    printf("\n===> Building RomFS\n");
    romfs_build(&settings->htmldoc_romfs_dir, &ivfc_lvls_path[5], &nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[5].hash_data_size);
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[5].block_size = 0x0E; // 0x4000

    // Create IVFC levels
    printf("\n===> Creating IVFC levels\n");
    for (int b = 4; b >= 0; b--)
    {
        printf("Writing %s\n", ivfc_lvls_path[b].char_path);
        ivfc_create_level(&ivfc_lvls_path[b], &ivfc_lvls_path[b + 1], &nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[b].hash_data_size);
        nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[b].block_size = 0x0E; // 0x4000
    }

    // Set IVFC levels logical offset
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[0].logical_offset = 0;
    for (int i = 1; i <= 5; i++)
        nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[i].logical_offset = nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[i - 1].logical_offset + nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[i - 1].hash_data_size;

    // Write IVFC levels
    printf("\n===> Writing IVFC levels\n");
    for (int c = 0; c < 6; c++)
    {
        printf("Writing %s to %s\n", ivfc_lvls_path[c].char_path, manual_htmldoc_nca_path.char_path);
        nca_write_file(manual_htmldoc_nca_file, &ivfc_lvls_path[c]);
    }

    // Write Padding if required
    nca_write_padding(manual_htmldoc_nca_file);

    // Common values
    nca_header.magic = MAGIC_NCA3;
    nca_header.content_type = 0x3; // Manual
    nca_header.sdk_version = settings->sdk_version;
    nca_header.title_id = cnmt_ctx->cnmt_header.title_id;
    nca_set_keygen(&nca_header, settings);

    nca_header.section_entries[0].media_start_offset = 0x6;                                                 // 0xC00 / 0x200
    nca_header.section_entries[0].media_end_offset = (uint32_t)(ftello64(manual_htmldoc_nca_file) / 0x200); // Section end offset / 200
    nca_header.section_entries[0]._0x8[0] = 0x1;                                                            // Always 1

    nca_header.fs_headers[0].hash_type = HASH_TYPE_ROMFS;
    nca_header.fs_headers[0].version = 0x2; // Always 2
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.magic = MAGIC_IVFC;
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.id = 0x20000; //Always 0x20000
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.master_hash_size = 0x20;
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.num_levels = 0x7;
    if (settings->plaintext == 0)
        nca_header.fs_headers[0].crypt_type = 0x3; // Regular crypto
    else
        nca_header.fs_headers[0].crypt_type = 0x1; // Plaintext

    // Calculate master hash and section hash
    printf("\n===> Calculating Hashes:\n");
    printf("Calculating Master hash\n");
    ivfc_calculate_master_hash(&ivfc_lvls_path[0], nca_header.fs_headers[0].romfs_superblock.ivfc_header.master_hash);
    printf("Calculating Section hash\n");
    nca_calculate_section_hash(&nca_header.fs_headers[0], nca_header.section_hashes[0]);

    printf("\n---> Finalizing:\n");

    // Set encrypted key area key 2
    memcpy(nca_header.encrypted_keys[2], settings->keyareakey, 0x10);

    printf("===> Encrypting NCA\n");
    if (settings->plaintext == 0)
    {
        // Encrypt section 0
        printf("Encrypting section 0\n");
        nca_encrypt_section(manual_htmldoc_nca_file, &nca_header, 0);
    }

    // Encrypt header
    printf("Getting NCA file size\n");
    fseeko64(manual_htmldoc_nca_file, 0, SEEK_END);
    nca_header.nca_size = (uint64_t)ftello64(manual_htmldoc_nca_file);
    printf("Encrypting key area\n");
    nca_encrypt_key_area(&nca_header, settings);
    printf("Encrypting header\n");
    nca_encrypt_header(&nca_header, settings);

    // Write MCA header
    printf("\n===> Writing NCA header\n");
    printf("Writing NCA header to %s\n", manual_htmldoc_nca_path.char_path);
    fseeko64(manual_htmldoc_nca_file, 0, SEEK_SET);
    fwrite(&nca_header, 1, sizeof(nca_header), manual_htmldoc_nca_file);

    // Calculate hash and nca size
    printf("\n===> Post creation process\n");
    printf("Calculating NCA hash\n");
    nca_calculate_hash(manual_htmldoc_nca_file, cnmt_ctx, 3);

    fclose(manual_htmldoc_nca_file);

    // Rename manual_htmldoc.nca to ncaid.nca
    filepath_t manual_htmldoc_nca_final_path;
    filepath_init(&manual_htmldoc_nca_final_path);
    filepath_copy(&manual_htmldoc_nca_final_path, &settings->nca_dir);
    char manual_htmldoc_nca_name[37];
    hexBinaryString(cnmt_ctx->cnmt_content_records[3].ncaid, 16, manual_htmldoc_nca_name, 33);
    strcat(manual_htmldoc_nca_name, ".nca");
    manual_htmldoc_nca_name[36] = '\0';
    printf("Renaming manual_htmldoc.nca to %s\n", manual_htmldoc_nca_name);
    filepath_append(&manual_htmldoc_nca_final_path, "%s", manual_htmldoc_nca_name);
    os_rename(manual_htmldoc_nca_path.os_path, manual_htmldoc_nca_final_path.os_path);
    printf("\n----> Created Manual(HtmlDoc) NCA: %s\n", manual_htmldoc_nca_final_path.char_path);
}

void nca_create_manual_legalinfo(hbp_settings_t *settings, cnmt_ctx_t *cnmt_ctx)
{
    printf("----> Creating Manual(LegalInfo) NCA:\n");
    printf("===> Creating NCA header\n");
    nca_header_t nca_header;
    memset(&nca_header, 0, sizeof(nca_header));

    filepath_t manual_legalinfo_nca_path;
    filepath_init(&manual_legalinfo_nca_path);
    filepath_copy(&manual_legalinfo_nca_path, &settings->nca_dir);
    filepath_append(&manual_legalinfo_nca_path, "manual_legalinfo.nca");

    FILE *manual_legalinfo_nca_file;
    manual_legalinfo_nca_file = os_fopen(manual_legalinfo_nca_path.os_path, OS_MODE_WRITE_EDIT);

    // Write placeholder for NCA header
    printf("Writing NCA header placeholder to %s\n", manual_legalinfo_nca_path.char_path);
    if (manual_legalinfo_nca_file != NULL)
        fwrite(&nca_header, 1, sizeof(nca_header), manual_legalinfo_nca_file);
    else
    {
        fprintf(stderr, "Failed to create %s!\n", manual_legalinfo_nca_path.char_path);
        exit(EXIT_FAILURE);
    }

    printf("\n---> Creating Section 0:");

    // Set IVFC levels temp filepaths
    filepath_t ivfc_lvls_path[6];
    for (int a = 0; a < 6; a++)
    {
        filepath_init(&ivfc_lvls_path[a]);
        filepath_copy(&ivfc_lvls_path[a], &settings->temp_dir);
        filepath_append(&ivfc_lvls_path[a], "manual_legalinfo_sec0_ivfc_lvl%i", a + 1);
    }

    //Build RomFS
    printf("\n===> Building RomFS\n");
    romfs_build(&settings->legalinfo_romfs_dir, &ivfc_lvls_path[5], &nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[5].hash_data_size);
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[5].block_size = 0x0E; // 0x4000

    // Create IVFC levels
    printf("\n===> Creating IVFC levels\n");
    for (int b = 4; b >= 0; b--)
    {
        printf("Writing %s\n", ivfc_lvls_path[b].char_path);
        ivfc_create_level(&ivfc_lvls_path[b], &ivfc_lvls_path[b + 1], &nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[b].hash_data_size);
        nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[b].block_size = 0x0E; // 0x4000
    }

    // Set IVFC levels logical offset
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[0].logical_offset = 0;
    for (int i = 1; i <= 5; i++)
        nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[i].logical_offset = nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[i - 1].logical_offset + nca_header.fs_headers[0].romfs_superblock.ivfc_header.level_headers[i - 1].hash_data_size;

    // Write IVFC levels
    printf("\n===> Writing IVFC levels\n");
    for (int c = 0; c < 6; c++)
    {
        printf("Writing %s to %s\n", ivfc_lvls_path[c].char_path, manual_legalinfo_nca_path.char_path);
        nca_write_file(manual_legalinfo_nca_file, &ivfc_lvls_path[c]);
    }

    // Write Padding if required
    nca_write_padding(manual_legalinfo_nca_file);

    // Common values
    nca_header.magic = MAGIC_NCA3;
    nca_header.content_type = 0x3; // Manual
    nca_header.sdk_version = settings->sdk_version;
    nca_header.title_id = cnmt_ctx->cnmt_header.title_id;
    nca_set_keygen(&nca_header, settings);

    nca_header.section_entries[0].media_start_offset = 0x6;                                                 // 0xC00 / 0x200
    nca_header.section_entries[0].media_end_offset = (uint32_t)(ftello64(manual_legalinfo_nca_file) / 0x200); // Section end offset / 200
    nca_header.section_entries[0]._0x8[0] = 0x1;                                                            // Always 1

    nca_header.fs_headers[0].hash_type = HASH_TYPE_ROMFS;
    nca_header.fs_headers[0].version = 0x2; // Always 2
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.magic = MAGIC_IVFC;
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.id = 0x20000; //Always 0x20000
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.master_hash_size = 0x20;
    nca_header.fs_headers[0].romfs_superblock.ivfc_header.num_levels = 0x7;
    if (settings->plaintext == 0)
        nca_header.fs_headers[0].crypt_type = 0x3; // Regular crypto
    else
        nca_header.fs_headers[0].crypt_type = 0x1; // Plaintext

    // Calculate master hash and section hash
    printf("\n===> Calculating Hashes:\n");
    printf("Calculating Master hash\n");
    ivfc_calculate_master_hash(&ivfc_lvls_path[0], nca_header.fs_headers[0].romfs_superblock.ivfc_header.master_hash);
    printf("Calculating Section hash\n");
    nca_calculate_section_hash(&nca_header.fs_headers[0], nca_header.section_hashes[0]);

    printf("\n---> Finalizing:\n");

    // Set encrypted key area key 2
    memcpy(nca_header.encrypted_keys[2], settings->keyareakey, 0x10);

    printf("===> Encrypting NCA\n");
    if (settings->plaintext == 0)
    {
        // Encrypt section 0
        printf("Encrypting section 0\n");
        nca_encrypt_section(manual_legalinfo_nca_file, &nca_header, 0);
    }

    // Encrypt header
    printf("Getting NCA file size\n");
    fseeko64(manual_legalinfo_nca_file, 0, SEEK_END);
    nca_header.nca_size = (uint64_t)ftello64(manual_legalinfo_nca_file);
    printf("Encrypting key area\n");
    nca_encrypt_key_area(&nca_header, settings);
    printf("Encrypting header\n");
    nca_encrypt_header(&nca_header, settings);

    // Write MCA header
    printf("\n===> Writing NCA header\n");
    printf("Writing NCA header to %s\n", manual_legalinfo_nca_path.char_path);
    fseeko64(manual_legalinfo_nca_file, 0, SEEK_SET);
    fwrite(&nca_header, 1, sizeof(nca_header), manual_legalinfo_nca_file);

    // Calculate hash and nca size
    printf("\n===> Post creation process\n");
    printf("Calculating NCA hash\n");
    nca_calculate_hash(manual_legalinfo_nca_file, cnmt_ctx, 4);

    fclose(manual_legalinfo_nca_file);

    // Rename manual_htmldoc.nca to ncaid.nca
    filepath_t manual_legalinfo_nca_final_path;
    filepath_init(&manual_legalinfo_nca_final_path);
    filepath_copy(&manual_legalinfo_nca_final_path, &settings->nca_dir);
    char manual_legalinfo_nca_name[37];
    hexBinaryString(cnmt_ctx->cnmt_content_records[4].ncaid, 16, manual_legalinfo_nca_name, 33);
    strcat(manual_legalinfo_nca_name, ".nca");
    manual_legalinfo_nca_name[36] = '\0';
    printf("Renaming manual_legalinfo.nca to %s\n", manual_legalinfo_nca_name);
    filepath_append(&manual_legalinfo_nca_final_path, "%s", manual_legalinfo_nca_name);
    os_rename(manual_legalinfo_nca_path.os_path, manual_legalinfo_nca_final_path.os_path);
    printf("\n----> Created Manual(LegalInfo) NCA: %s\n", manual_legalinfo_nca_final_path.char_path);
}

void nca_create_program(hbp_settings_t *settings, cnmt_ctx_t *cnmt_ctx)
{
    printf("----> Creating Program NCA:\n");
    printf("===> Creating NCA header\n");
    nca_header_t nca_header;
    memset(&nca_header, 0, sizeof(nca_header));

    filepath_t program_nca_path;
    filepath_init(&program_nca_path);
    filepath_copy(&program_nca_path, &settings->nca_dir);
    filepath_append(&program_nca_path, "program.nca");

    FILE *program_nca_file;
    program_nca_file = os_fopen(program_nca_path.os_path, OS_MODE_WRITE_EDIT);

    // Write placeholder for NCA header
    printf("Writing NCA header placeholder to %s\n", program_nca_path.char_path);
    if (program_nca_file != NULL)
        fwrite(&nca_header, 1, sizeof(nca_header), program_nca_file);
    else
    {
        fprintf(stderr, "Failed to create %s!\n", program_nca_path.char_path);
        exit(EXIT_FAILURE);
    }

    printf("\n---> Creating Section 0:");

    //Build ExeFS
    filepath_t program_exefs;
    filepath_init(&program_exefs);
    filepath_copy(&program_exefs, &settings->temp_dir);
    filepath_append(&program_exefs, "program_sec0_exefs");
    filepath_t program_exefs_hash_table;
    filepath_init(&program_exefs_hash_table);
    filepath_copy(&program_exefs_hash_table, &settings->temp_dir);
    filepath_append(&program_exefs_hash_table, "program_sec0_exefs_hashtable");
    uint32_t exefs_hash_block_size = PFS0_EXEFS_HASH_BLOCK_SIZE;
    printf("\n===> Building ExeFS\n");
    pfs0_build(&settings->exefs_dir, &program_exefs, &nca_header.fs_headers[0].pfs0_superblock.pfs0_size);
    printf("Calculating hash table\n");
    pfs0_create_hashtable(&program_exefs, &program_exefs_hash_table, exefs_hash_block_size, &nca_header.fs_headers[0].pfs0_superblock.hash_table_size, &nca_header.fs_headers[0].pfs0_superblock.pfs0_offset);

    // Write ExeFS
    printf("\n===> Writing ExeFS\n");
    printf("Writing PFS0 hash table\n");
    nca_write_file(program_nca_file, &program_exefs_hash_table);
    printf("Writing PFS0\n");
    nca_write_file(program_nca_file, &program_exefs);

    // Write Padding if required
    nca_write_padding(program_nca_file);

    // Common values
    nca_header.magic = MAGIC_NCA3;
    nca_header.content_type = 0x0; // Program
    nca_header.sdk_version = settings->sdk_version;
    nca_header.title_id = cnmt_ctx->cnmt_header.title_id;
    nca_set_keygen(&nca_header, settings);

    nca_header.section_entries[0].media_start_offset = 0x6;                                          // 0xC00 / 0x200
    nca_header.section_entries[0].media_end_offset = (uint32_t)(ftello64(program_nca_file) / 0x200); // Section end offset / 200
    nca_header.section_entries[0]._0x8[0] = 0x1;                                                     // Always 1

    nca_header.fs_headers[0].hash_type = HASH_TYPE_PFS0;
    nca_header.fs_headers[0].fs_type = FS_TYPE_PFS0;
    nca_header.fs_headers[0].version = 0x2; // Always 2
    nca_header.fs_headers[0].pfs0_superblock.always_2 = 0x2;
    nca_header.fs_headers[0].pfs0_superblock.block_size = exefs_hash_block_size;
    if (settings->plaintext == 0)
        nca_header.fs_headers[0].crypt_type = 0x3; // Regular crypto
    else
        nca_header.fs_headers[0].crypt_type = 0x1; // Plaintext

    // Calculate master hash and section hash
    printf("\n===> Calculating Hashes:\n");
    printf("Calculating Master hash\n");
    pfs0_calculate_master_hash(&program_exefs_hash_table, nca_header.fs_headers[0].pfs0_superblock.hash_table_size, nca_header.fs_headers[0].pfs0_superblock.master_hash);
    printf("Calculating Section hash\n");
    nca_calculate_section_hash(&nca_header.fs_headers[0], nca_header.section_hashes[0]);

    if (settings->noromfs == 0)
    {

        printf("\n---> Creating Section 1:");

        // Set IVFC levels temp filepaths
        filepath_t ivfc_lvls_path[6];
        for (int a = 0; a < 6; a++)
        {
            filepath_init(&ivfc_lvls_path[a]);
            filepath_copy(&ivfc_lvls_path[a], &settings->temp_dir);
            filepath_append(&ivfc_lvls_path[a], "program_sec1_ivfc_lvl%i", a + 1);
        }

        //Build RomFS
        printf("\n===> Building RomFS\n");
        romfs_build(&settings->romfs_dir, &ivfc_lvls_path[5], &nca_header.fs_headers[1].romfs_superblock.ivfc_header.level_headers[5].hash_data_size);
        nca_header.fs_headers[1].romfs_superblock.ivfc_header.level_headers[5].block_size = 0x0E; // 0x4000

        // Create IVFC levels
        printf("\n===> Creating IVFC levels\n");
        for (int b = 4; b >= 0; b--)
        {
            printf("Writing %s\n", ivfc_lvls_path[b].char_path);
            ivfc_create_level(&ivfc_lvls_path[b], &ivfc_lvls_path[b + 1], &nca_header.fs_headers[1].romfs_superblock.ivfc_header.level_headers[b].hash_data_size);
            nca_header.fs_headers[1].romfs_superblock.ivfc_header.level_headers[b].block_size = 0x0E; // 0x4000
        }

        // Set IVFC levels logical offset
        nca_header.fs_headers[1].romfs_superblock.ivfc_header.level_headers[0].logical_offset = 0;
        for (int i = 1; i <= 5; i++)
            nca_header.fs_headers[1].romfs_superblock.ivfc_header.level_headers[i].logical_offset = nca_header.fs_headers[1].romfs_superblock.ivfc_header.level_headers[i - 1].logical_offset + nca_header.fs_headers[1].romfs_superblock.ivfc_header.level_headers[i - 1].hash_data_size;

        // Write IVFC levels
        printf("\n===> Writing IVFC levels\n");
        for (int c = 0; c < 6; c++)
        {
            printf("Writing %s to %s\n", ivfc_lvls_path[c].char_path, program_nca_path.char_path);
            nca_write_file(program_nca_file, &ivfc_lvls_path[c]);
        }

        // Write Padding if required
        nca_write_padding(program_nca_file);

        // Set header values
        nca_header.section_entries[1].media_start_offset = nca_header.section_entries[0].media_end_offset;
        nca_header.section_entries[1].media_end_offset = (uint32_t)(ftello64(program_nca_file) / 0x200);
        nca_header.section_entries[1]._0x8[0] = 0x1; // Always 1

        nca_header.fs_headers[1].hash_type = HASH_TYPE_ROMFS;
        nca_header.fs_headers[1].version = 0x2; // Always 2
        nca_header.fs_headers[1].romfs_superblock.ivfc_header.magic = MAGIC_IVFC;
        nca_header.fs_headers[1].romfs_superblock.ivfc_header.id = 0x20000; //Always 0x20000
        nca_header.fs_headers[1].romfs_superblock.ivfc_header.master_hash_size = 0x20;
        nca_header.fs_headers[1].romfs_superblock.ivfc_header.num_levels = 0x7;
        if (settings->plaintext == 0)
            nca_header.fs_headers[1].crypt_type = 0x3; // Regular crypto
        else
            nca_header.fs_headers[1].crypt_type = 0x1; // Plaintext

        // Calculate master hash and section hash
        printf("\n===> Calculating Hashes:\n");
        printf("Calculating Master hash\n");
        ivfc_calculate_master_hash(&ivfc_lvls_path[0], nca_header.fs_headers[1].romfs_superblock.ivfc_header.master_hash);
        printf("Calculating Section hash\n");
        nca_calculate_section_hash(&nca_header.fs_headers[1], nca_header.section_hashes[1]);
    }

    if (settings->nologo == 0)
    {
        printf("\n---> Creating Section 2:");

        //Build logo
        filepath_t program_logo;
        filepath_init(&program_logo);
        filepath_copy(&program_logo, &settings->temp_dir);
        filepath_append(&program_logo, "program_sec2_logo");
        filepath_t program_logo_hash_table;
        filepath_init(&program_logo_hash_table);
        filepath_copy(&program_logo_hash_table, &settings->temp_dir);
        filepath_append(&program_logo_hash_table, "program_sec2_logo_hashtable");
        uint32_t logo_hash_block_size = PFS0_LOGO_HASH_BLOCK_SIZE;
        printf("\n===> Building PFS0\n");
        pfs0_build(&settings->logo_dir, &program_logo, &nca_header.fs_headers[2].pfs0_superblock.pfs0_size);
        printf("Calculating hash table\n");
        pfs0_create_hashtable(&program_logo, &program_logo_hash_table, logo_hash_block_size, &nca_header.fs_headers[2].pfs0_superblock.hash_table_size, &nca_header.fs_headers[2].pfs0_superblock.pfs0_offset);

        // Write IVFC levels
        printf("\n===> Writing IVFC levels\n");
        printf("Writing PFS0 hash table\n");
        nca_write_file(program_nca_file, &program_logo_hash_table);
        printf("Writing PFS0\n");
        nca_write_file(program_nca_file, &program_logo);

        // Write Padding if required
        nca_write_padding(program_nca_file);

        if (settings->noromfs == 0)
            nca_header.section_entries[2].media_start_offset = nca_header.section_entries[1].media_end_offset;
        else
            nca_header.section_entries[2].media_start_offset = nca_header.section_entries[0].media_end_offset;

        nca_header.section_entries[2].media_end_offset = (uint32_t)(ftello64(program_nca_file) / 0x200); // Section end offset / 200
        nca_header.section_entries[2]._0x8[0] = 0x1;                                                     // Always 1

        nca_header.fs_headers[2].hash_type = HASH_TYPE_PFS0;
        nca_header.fs_headers[2].fs_type = FS_TYPE_PFS0;
        nca_header.fs_headers[2].version = 0x2;    // Always 2
        nca_header.fs_headers[2].crypt_type = 0x1; // Plain text
        nca_header.fs_headers[2].pfs0_superblock.always_2 = 0x2;
        nca_header.fs_headers[2].pfs0_superblock.block_size = logo_hash_block_size;

        // Calculate master hash and section hash
        printf("\n===> Calculating Hashes:\n");
        printf("Calculating Master hash\n");
        pfs0_calculate_master_hash(&program_logo_hash_table, nca_header.fs_headers[2].pfs0_superblock.hash_table_size, nca_header.fs_headers[2].pfs0_superblock.master_hash);
        printf("Calculating Section hash\n");
        nca_calculate_section_hash(&nca_header.fs_headers[2], nca_header.section_hashes[2]);
    }

    printf("\n---> Finalizing:\n");

    // Set encrypted key area key 2
    memcpy(nca_header.encrypted_keys[2], settings->keyareakey, 0x10);

    printf("===> Encrypting NCA\n");
    // Encrypt sections
    if (settings->plaintext == 0)
    {
        printf("Encrypting section 0\n");
        nca_encrypt_section(program_nca_file, &nca_header, 0);
        if (settings->noromfs == 0)
        {
            printf("Encrypting section 1\n");
            nca_encrypt_section(program_nca_file, &nca_header, 1);
        }
    }

    // Encrypt header
    printf("Getting NCA file size\n");
    fseeko64(program_nca_file, 0, SEEK_END);
    nca_header.nca_size = (uint64_t)ftello64(program_nca_file);
    printf("Encrypting key area\n");
    nca_encrypt_key_area(&nca_header, settings);
    if (settings->nosignncasig2 == 0)
    {
        printf("Signing nca header\n");
        rsa_sign(&nca_header.magic, 0x200, (unsigned char *)&nca_header.npdm_key_sig, 0x100);
    }
    printf("Encrypting header\n");
    nca_encrypt_header(&nca_header, settings);

    // Write MCA header
    printf("\n===> Writing NCA header\n");
    printf("Writing NCA header to %s\n", program_nca_path.char_path);
    fseeko64(program_nca_file, 0, SEEK_SET);
    fwrite(&nca_header, 1, sizeof(nca_header), program_nca_file);

    // Calculate hash and nca size
    printf("\n===> Post creation process\n");
    printf("Calculating NCA hash\n");
    nca_calculate_hash(program_nca_file, cnmt_ctx, 0);

    fclose(program_nca_file);

    // Rename program.nca to ncaid.nca
    filepath_t program_nca_final_path;
    filepath_init(&program_nca_final_path);
    filepath_copy(&program_nca_final_path, &settings->nca_dir);
    char program_nca_name[37];
    hexBinaryString(cnmt_ctx->cnmt_content_records[0].ncaid, 16, program_nca_name, 33);
    strcat(program_nca_name, ".nca");
    program_nca_name[36] = '\0';
    printf("Renaming program.nca to %s\n", program_nca_name);
    filepath_append(&program_nca_final_path, "%s", program_nca_name);
    os_rename(program_nca_path.os_path, program_nca_final_path.os_path);
    printf("\n----> Created Program NCA: %s\n", program_nca_final_path.char_path);
}

void nca_create_meta(hbp_settings_t *settings, cnmt_ctx_t *cnmt_ctx)
{
    printf("----> Creating Metadata NCA:\n");
    printf("===> Creating NCA header\n");
    nca_header_t nca_header;
    memset(&nca_header, 0, sizeof(nca_header));

    filepath_t meta_nca_path;
    filepath_init(&meta_nca_path);
    filepath_copy(&meta_nca_path, &settings->nca_dir);
    filepath_append(&meta_nca_path, "meta.nca");

    FILE *meta_nca_file;
    meta_nca_file = os_fopen(meta_nca_path.os_path, OS_MODE_WRITE_EDIT);

    // Write placeholder for NCA header
    printf("Writing NCA header placeholder to %s\n", meta_nca_path.char_path);
    if (meta_nca_file != NULL)
        fwrite(&nca_header, 1, sizeof(nca_header), meta_nca_file);
    else
    {
        fprintf(stderr, "Failed to create %s!\n", meta_nca_path.char_path);
        exit(EXIT_FAILURE);
    }

    filepath_t cnmt_path;
    filepath_init(&cnmt_path);
    filepath_copy(&cnmt_path, &settings->temp_dir);
    filepath_append(&cnmt_path, "cnmt");
    filepath_t cnmt_dir_path;
    filepath_init(&cnmt_dir_path);
    filepath_copy(&cnmt_dir_path, &cnmt_path);
    // Create cnmt directory if required
    os_makedir(cnmt_dir_path.os_path);
    // Cnmt filename = Application_tid.cnmt
    filepath_append(&cnmt_path, "Application_%016" PRIx64 ".cnmt", cnmt_ctx->cnmt_header.title_id);
    printf("\n===> Creating Metadata file\n");
    cnmt_create(cnmt_ctx, &cnmt_path, settings);

    //Build PFS0
    filepath_t meta_pfs0;
    filepath_init(&meta_pfs0);
    filepath_copy(&meta_pfs0, &settings->temp_dir);
    filepath_append(&meta_pfs0, "meta_sec0_pfs0");
    filepath_t meta_pfs0_hash_table;
    filepath_init(&meta_pfs0_hash_table);
    filepath_copy(&meta_pfs0_hash_table, &settings->temp_dir);
    filepath_append(&meta_pfs0_hash_table, "meta_sec0_pfs0_hashtable");
    uint32_t meta_hash_block_size = PFS0_META_HASH_BLOCK_SIZE;
    printf("\n===> Building PFS0\n");
    pfs0_build(&cnmt_dir_path, &meta_pfs0, &nca_header.fs_headers[0].pfs0_superblock.pfs0_size);
    printf("Calculating hash table\n");
    pfs0_create_hashtable(&meta_pfs0, &meta_pfs0_hash_table, meta_hash_block_size, &nca_header.fs_headers[0].pfs0_superblock.hash_table_size, &nca_header.fs_headers[0].pfs0_superblock.pfs0_offset);

    // Write ExeFS
    printf("\n===> Writing PFS0 section\n");
    printf("Writing PFS0 hash table\n");
    nca_write_file(meta_nca_file, &meta_pfs0_hash_table);
    printf("Writing PFS0\n");
    nca_write_file(meta_nca_file, &meta_pfs0);

    // Write Padding if required
    nca_write_padding(meta_nca_file);

    // Common values
    nca_header.magic = MAGIC_NCA3;
    nca_header.content_type = 0x1; // Meta
    nca_header.sdk_version = settings->sdk_version;
    nca_header.title_id = cnmt_ctx->cnmt_header.title_id;
    nca_set_keygen(&nca_header, settings);

    nca_header.section_entries[0].media_start_offset = 0x6;                                       // 0xC00 / 0x200
    nca_header.section_entries[0].media_end_offset = (uint32_t)(ftello64(meta_nca_file) / 0x200); // Section end offset / 200
    nca_header.section_entries[0]._0x8[0] = 0x1;                                                  // Always 1

    nca_header.fs_headers[0].hash_type = HASH_TYPE_PFS0;
    nca_header.fs_headers[0].fs_type = FS_TYPE_PFS0;
    nca_header.fs_headers[0].version = 0x2; // Always 2
    nca_header.fs_headers[0].pfs0_superblock.always_2 = 0x2;
    nca_header.fs_headers[0].pfs0_superblock.block_size = meta_hash_block_size;
    if (settings->plaintext == 0)
        nca_header.fs_headers[0].crypt_type = 0x3; // Regular crypto
    else
        nca_header.fs_headers[0].crypt_type = 0x1; // Plaintext

    // Calculate master hash and section hash
    printf("\n===> Calculating Hashes:\n");
    printf("Calculating Master hash\n");
    pfs0_calculate_master_hash(&meta_pfs0_hash_table, nca_header.fs_headers[0].pfs0_superblock.hash_table_size, nca_header.fs_headers[0].pfs0_superblock.master_hash);
    printf("Calculating Section hash\n");
    nca_calculate_section_hash(&nca_header.fs_headers[0], nca_header.section_hashes[0]);

    printf("\n---> Finalizing:\n");

    // Set encrypted key area key 2
    memcpy(nca_header.encrypted_keys[2], settings->keyareakey, 0x10);

    printf("===> Encrypting NCA\n");
    if (settings->plaintext == 0)
    {
        // Encrypt section 0
        printf("Encrypting section 0\n");
        nca_encrypt_section(meta_nca_file, &nca_header, 0);
    }

    // Encrypt header
    printf("Getting NCA file size\n");
    fseeko64(meta_nca_file, 0, SEEK_END);
    nca_header.nca_size = (uint64_t)ftello64(meta_nca_file);
    printf("Encrypting key area\n");
    nca_encrypt_key_area(&nca_header, settings);
    printf("Encrypting header\n");
    nca_encrypt_header(&nca_header, settings);

    // Write MCA header
    printf("\n===> Writing NCA header\n");
    printf("Writing NCA header to %s\n", meta_nca_path.char_path);
    fseeko64(meta_nca_file, 0, SEEK_SET);
    fwrite(&nca_header, 1, sizeof(nca_header), meta_nca_file);

    // Calculate hash and nca size
    printf("\n===> Post creation process\n");
    printf("Calculating NCA hash\n");
    nca_calculate_hash(meta_nca_file, cnmt_ctx, 2);

    fclose(meta_nca_file);

    // Rename meta.nca to ncaid.cnmt.nca
    filepath_t meta_nca_final_path;
    filepath_init(&meta_nca_final_path);
    filepath_copy(&meta_nca_final_path, &settings->nca_dir);
    char meta_nca_name[42];
    hexBinaryString(cnmt_ctx->cnmt_content_records[2].ncaid, 16, meta_nca_name, 33);
    strcat(meta_nca_name, ".cnmt.nca");
    meta_nca_name[41] = '\0';
    printf("Renaming meta.nca to %s\n", meta_nca_name);
    filepath_append(&meta_nca_final_path, "%s", meta_nca_name);
    os_rename(meta_nca_path.os_path, meta_nca_final_path.os_path);
    printf("\n----> Created Metadata NCA: %s\n", meta_nca_final_path.char_path);
}

void nca_write_file(FILE *nca_file, filepath_t *file_path)
{
    uint64_t file_size;
    FILE *fl;
    fl = os_fopen(file_path->os_path, OS_MODE_READ);

    if (fl == NULL)
    {
        fprintf(stderr, "Failed to open %s!\n", file_path->char_path);
        exit(EXIT_FAILURE);
    }

    // Get IVFC level file filesize
    fseeko64(fl, 0, SEEK_END);
    file_size = ftello64(fl);
    fseeko64(fl, 0, SEEK_SET);

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
        if (fread(buf, 1, read_size, fl) != read_size)
        {
            fprintf(stderr, "Failed to read file %s\n", file_path->char_path);
            exit(EXIT_FAILURE);
        }
        fwrite(buf, read_size, 1, nca_file);
        ofs += read_size;
    }

    free(buf);
    fclose(fl);
}

// Write padding for media_end_offset
void nca_write_padding(FILE *nca_file)
{
    unsigned char *buf = (unsigned char *)calloc(1, 0x200);
    uint64_t curr_offset = ftello64(nca_file);
    uint64_t block_size = 0x200;
    uint64_t padding_size = block_size - (curr_offset % block_size);
    if (curr_offset % block_size != 0)
        fwrite(buf, 1, padding_size, nca_file);
    free(buf);
}

void nca_calculate_section_hash(nca_fs_header_t *fs_header, uint8_t *out_section_hash)
{
    // Calculate hash
    sha_ctx_t *sha_ctx = new_sha_ctx(HASH_TYPE_SHA256, 0);
    sha_update(sha_ctx, fs_header, 0x200);
    sha_get_hash(sha_ctx, (unsigned char *)out_section_hash);
    free_sha_ctx(sha_ctx);
}

void nca_encrypt_key_area(nca_header_t *nca_header, hbp_settings_t *settings)
{
    aes_ctx_t *aes_ctx = new_aes_ctx(settings->keyset.key_area_keys[settings->keygeneration - 1][0], 16, AES_MODE_ECB);
    aes_encrypt(aes_ctx, nca_header->encrypted_keys, nca_header->encrypted_keys, 0x40);
    free_aes_ctx(aes_ctx);
}

void nca_encrypt_header(nca_header_t *nca_header, hbp_settings_t *settings)
{
    aes_ctx_t *hdr_aes_ctx = new_aes_ctx(settings->keyset.header_key, 32, AES_MODE_XTS);
    aes_xts_encrypt(hdr_aes_ctx, nca_header, nca_header, 0xC00, 0, 0x200);
    free_aes_ctx(hdr_aes_ctx);
}

void nca_encrypt_section(FILE *nca_file, nca_header_t *nca_header, uint8_t section_index)
{
    uint64_t start_offset = nca_header->section_entries[section_index].media_start_offset;
    start_offset *= 0x200;
    uint64_t end_offset = nca_header->section_entries[section_index].media_end_offset;
    end_offset *= 0x200;
    uint64_t filesize = end_offset - start_offset;

    // Calculate counter for section encryption
    uint64_t ctr_ofs = start_offset >> 4;
    unsigned char ctr[0x10] = {0};
    for (unsigned int j = 0; j < 0x8; j++)
    {
        ctr[j] = nca_header->fs_headers[section_index].section_ctr[0x8 - j - 1];
        ctr[0x10 - j - 1] = (unsigned char)(ctr_ofs & 0xFF);
        ctr_ofs >>= 8;
    }

    uint64_t read_size = 0x6000000; // ~100 MB buffer.
    unsigned char *buf = malloc(read_size);
    if (buf == NULL)
    {
        fprintf(stderr, "Failed to allocate file-read buffer!\n");
        exit(EXIT_FAILURE);
    }
    uint64_t ofs = 0;
    fseeko64(nca_file, start_offset, SEEK_SET);
    aes_ctx_t *aes_ctx = new_aes_ctx(nca_header->encrypted_keys[2], 16, AES_MODE_CTR);
    while (ofs < filesize)
    {
        if (ofs + read_size >= filesize)
            read_size = filesize - ofs;
        if (fread(buf, 1, read_size, nca_file) != read_size)
        {
            fprintf(stderr, "Failed to read file!\n");
            exit(EXIT_FAILURE);
        }
        fseeko64(nca_file, start_offset + ofs, SEEK_SET);
        aes_setiv(aes_ctx, ctr, 0x10);
        aes_encrypt(aes_ctx, buf, buf, read_size);
        fwrite(buf, 1, read_size, nca_file);
        ofs += read_size;
        nca_update_ctr(ctr, start_offset + ofs);
    }

    free(buf);
    free_aes_ctx(aes_ctx);
}

/* Updates the CTR for an offset. */
void nca_update_ctr(unsigned char *ctr, uint64_t ofs)
{
    ofs >>= 4;
    for (unsigned int j = 0; j < 0x8; j++)
    {
        ctr[0x10 - j - 1] = (unsigned char)(ofs & 0xFF);
        ofs >>= 8;
    }
}

void nca_calculate_hash(FILE *nca_file, cnmt_ctx_t *cnmt_ctx, uint8_t cnmt_index)
{
    uint64_t file_size;
    // Get source file size
    fseeko64(nca_file, 0, SEEK_END);
    file_size = (uint64_t)ftello64(nca_file);
    memcpy(cnmt_ctx->cnmt_content_records[cnmt_index].size, &file_size, 0x6);

    sha_ctx_t *sha_ctx = new_sha_ctx(HASH_TYPE_SHA256, 0);
    uint64_t read_size = 0x61A8000; // 100 MB buffer.
    unsigned char *buf = malloc(read_size);
    fseeko64(nca_file, 0, SEEK_SET);

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
        if (fread(buf, 1, read_size, nca_file) != read_size)
        {
            fprintf(stderr, "Failed to read file!\n");
            exit(EXIT_FAILURE);
        }
        sha_update(sha_ctx, buf, read_size);
        ofs += read_size;
    }
    sha_get_hash(sha_ctx, cnmt_ctx->cnmt_content_records[cnmt_index].hash);
    memcpy(cnmt_ctx->cnmt_content_records[cnmt_index].ncaid, &cnmt_ctx->cnmt_content_records[cnmt_index].hash, 0x10); // NcaID = first 16 bytes of hash

    free(buf);
    free_sha_ctx(sha_ctx);
}

void nca_set_keygen(nca_header_t *nca_header, hbp_settings_t *settings)
{
    if (settings->keygeneration != 1)
    {
        if (settings->keygeneration == 2)
            nca_header->crypto_type = 0x2;
        else
        {
            nca_header->crypto_type = 0x2;
            nca_header->crypto_type2 = settings->keygeneration;
        }
    }
}