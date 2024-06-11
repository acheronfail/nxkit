#include <getopt.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "nca.h"
#include "utils.h"
#include "settings.h"
#include "pki.h"
#include "extkeys.h"
#include "version.h"
#include "nacp.h"
#include "npdm.h"
#include "cnmt.h"
#include "pfs0.h"

/* hacBrewPack by The-4n */

// Print Usage
static void usage(void)
{
    fprintf(stderr,
            "Usage: %s [options...]\n\n"
            "Options:\n"
            "-k, --keyset             Set keyset filepath, default filepath is ." OS_PATH_SEPARATOR "keys.dat\n"
            "-h, --help               Display usage\n"
            "--nspdir                 Set output nsp directory path, default path is ." OS_PATH_SEPARATOR "hacbrewpack_nsp" OS_PATH_SEPARATOR "\n"
            "--ncadir                 Set output nca directory path, default path is ." OS_PATH_SEPARATOR "hacbrewpack_nca" OS_PATH_SEPARATOR "\n"
            "--tempdir                Set temp directory filepath, default filepath is ." OS_PATH_SEPARATOR "hacbrewpack_temp" OS_PATH_SEPARATOR "\n"
            "--backupdir              Set backup directory path, default path is ." OS_PATH_SEPARATOR "hacbrewpack_backup" OS_PATH_SEPARATOR "\n"
            "--exefsdir               Set program exefs directory path, default path is ." OS_PATH_SEPARATOR "exefs" OS_PATH_SEPARATOR "\n"
            "--romfsdir               Set program romfs directory path, default path is ." OS_PATH_SEPARATOR "romfs" OS_PATH_SEPARATOR "\n"
            "--logodir                Set program logo directory path, default path is ." OS_PATH_SEPARATOR "logo" OS_PATH_SEPARATOR "\n"
            "--controldir             Set control romfs directory path, default path is ." OS_PATH_SEPARATOR "control" OS_PATH_SEPARATOR "\n"
            "--htmldocdir             Set HtmlDocument romfs directory path\n"
            "--legalinfodir           Set LegalInformation romfs directory path\n"
            "--noromfs                Skip creating program romfs section\n"
            "--nologo                 Skip creating program logo section\n"
            "--keygeneration          Set keygeneration for encrypting key area, default keygeneration is 1\n"
            "--keyareakey             Set key area key 2 in hex with 16 bytes lenght\n"
            "--sdkversion             Set SDK version in hex, default SDK version is 000C1100\n"
            "--plaintext              Skip encrypting sections and set section header block crypto type to plaintext\n"
            "--keepncadir             Keep NCA directory\n"
            "--nosignncasig2          Skip patching acid public key in npdm and signing nca header with acid public key\n"
            "Overriding options:\n"
            "--titleid                Use specified titleid for creating ncas and patch titleid in npdm and nacp\n"
            "--titlename              Change title name in nacp for all languages, max size is 512 bytes\n"
            "--titlepublisher         Change title publisher in nacp for all languages, max size is 256 bytes\n"
            "--nopatchnacplogo        Skip changing logo handeling to auto in NACP\n",
            USAGE_PROGRAM_NAME);
    exit(EXIT_FAILURE);
}

int main(int argc, char **argv)
{
    hbp_settings_t settings;
    cnmt_ctx_t cnmt_ctx;
    memset(&settings, 0, sizeof(settings));
    memset(&cnmt_ctx, 0, sizeof(cnmt_ctx));

    printf("hacBrewPack %s by The-4n\n\n", HACBREWPACK_VERSION);

    // Hardcode default temp directory
    filepath_init(&settings.temp_dir);
    filepath_set(&settings.temp_dir, "hacbrewpack_temp");

    // Hardcode default output nca directory
    filepath_init(&settings.nca_dir);
    filepath_set(&settings.nca_dir, "hacbrewpack_nca");

    // Hardcode default output nsp directory
    filepath_init(&settings.nsp_dir);
    filepath_set(&settings.nsp_dir, "hacbrewpack_nsp");

    // Hardcode default backup directory
    filepath_init(&settings.backup_dir);
    filepath_set(&settings.backup_dir, "hacbrewpack_backup");

    // Hardcode Program exeFS directory
    filepath_init(&settings.exefs_dir);
    filepath_set(&settings.exefs_dir, "exefs");

    // Hardcode Program RomFS directory
    filepath_init(&settings.romfs_dir);
    filepath_set(&settings.romfs_dir, "romfs");

    // Hardcode Program Logo directory
    filepath_init(&settings.logo_dir);
    filepath_set(&settings.logo_dir, "logo");

    // Hardcode Control RomFS directory
    filepath_init(&settings.control_romfs_dir);
    filepath_set(&settings.control_romfs_dir, "control");

    filepath_t keypath;
    filepath_init(&keypath);

    pki_initialize_keyset(&settings.keyset);

    // Default Settings
    settings.keygeneration = 1;
    settings.sdk_version = 0x000C1100;
    settings.keyareakey = (unsigned char *)calloc(1, 0x10);
    memset(settings.keyareakey, 4, 0x10);

    // Parse options
    while (1)
    {
        int option_index;
        int c;
        static struct option long_options[] =
            {
                {"keyset", 1, NULL, 'k'},
                {"help", 0, NULL, 'h'},
                {"tempdir", 1, NULL, 1},
                {"ncadir", 1, NULL, 2},
                {"nspdir", 1, NULL, 3},
                {"exefsdir", 1, NULL, 4},
                {"romfsdir", 1, NULL, 5},
                {"logodir", 1, NULL, 6},
                {"controldir", 1, NULL, 7},
                {"noromfs", 0, NULL, 8},
                {"nologo", 0, NULL, 9},
                {"plaintext", 0, NULL, 10},
                {"keygeneration", 1, NULL, 11},
                {"sdkversion", 1, NULL, 12},
                {"keyareakey", 1, NULL, 13},
                {"keepncadir", 0, NULL, 14},
                {"nopatchnacplogo", 0, NULL, 15},
                {"titleid", 1, NULL, 16},
                {"titlename", 1, NULL, 17},
                {"titlepublisher", 1, NULL, 18},
                {"htmldocdir", 1, NULL, 19},
                {"nosignncasig2", 0, NULL, 20},
                {"legalinfodir", 1, NULL, 21},
                {"backupdir", 1, NULL, 22},
                {NULL, 0, NULL, 0},
            };

        c = getopt_long(argc, argv, "k:h", long_options, &option_index);
        if (c == -1)
            break;

        switch (c)
        {
        case 'k':
            filepath_set(&keypath, optarg);
            break;
        case 'h':
            usage();
            break;
        case 1:
            filepath_set(&settings.temp_dir, optarg);
            break;
        case 2:
            filepath_set(&settings.nca_dir, optarg);
            break;
        case 3:
            filepath_set(&settings.nsp_dir, optarg);
            break;
        case 4:
            filepath_set(&settings.exefs_dir, optarg);
            break;
        case 5:
            filepath_set(&settings.romfs_dir, optarg);
            break;
        case 6:
            filepath_set(&settings.logo_dir, optarg);
            break;
        case 7:
            filepath_set(&settings.control_romfs_dir, optarg);
            break;
        case 8:
            settings.noromfs = 1;
            break;
        case 9:
            settings.nologo = 1;
            break;
        case 10:
            settings.plaintext = 1;
            break;
        case 11:
            settings.keygeneration = atoi(optarg);
            // Validating Keygeneration
            if (settings.keygeneration < 1 || settings.keygeneration > 32)
            {
                fprintf(stderr, "Invalid keygeneration: %i, keygeneration range: 1-32\n", settings.keygeneration);
                return EXIT_FAILURE;
            }
            break;
        case 12:
            settings.sdk_version = strtoul(optarg, NULL, 16);
            // Validating SDK Version
            if (settings.sdk_version < 0x000B0000)
            {
                fprintf(stderr, "Error: Invalid SDK version: %08" PRIX32 "\n"
                                "Valid SDK version range: 000B0000 - 00FFFFFF\n",
                        settings.sdk_version);
                exit(EXIT_FAILURE);
            }
            break;
        case 13:
            parse_hex_key(settings.keyareakey, optarg, 0x10);
            break;
        case 14:
            settings.keepncadir = 1;
            break;
        case 15:
            settings.nopatchnacplogo = 1;
            break;
        case 16:
            settings.title_id = strtoull(optarg, NULL, 16);
            break;
        case 17:
            if (strlen(optarg) < 0x200)
                strcpy(settings.titlename, optarg);
            else
            {
                fprintf(stderr, "Error: titlename is longer than 512 bytes\n");
                exit(EXIT_FAILURE);
            }
            break;
        case 18:
            if (strlen(optarg) < 0x100)
                strcpy(settings.titlepublisher, optarg);
            else
            {
                fprintf(stderr, "Error: titlepublisher is longer than 256 bytes\n");
                exit(EXIT_FAILURE);
            }
            break;
        case 19:
            filepath_init(&settings.htmldoc_romfs_dir);
            filepath_set(&settings.htmldoc_romfs_dir, optarg);
            break;
        case 20:
            settings.nosignncasig2 = 1;
            break;
        case 21:
            filepath_init(&settings.legalinfo_romfs_dir);
            filepath_set(&settings.legalinfo_romfs_dir, optarg);
            break;
        case 22:
            filepath_set(&settings.backup_dir, optarg);
            break;
        default:
            usage();
        }
    }

    // Remove existing temp and nca directories and Create new ones + nsp directory
    printf("Removing existing temp and nca directories\n");
    filepath_remove_directory(&settings.temp_dir);
    filepath_remove_directory(&settings.nca_dir);
    printf("Creating temp, nca, nsp and backup directories\n");
    os_makedir(settings.temp_dir.os_path);
    os_makedir(settings.nca_dir.os_path);
    os_makedir(settings.nsp_dir.os_path);
    os_makedir(settings.backup_dir.os_path);

    // Try to populate default keyfile.
    FILE *keyfile = NULL;
    if (keypath.valid == VALIDITY_INVALID)
    {
        // Locating default key file
        filepath_set(&keypath, "keys.dat");
        keyfile = os_fopen(keypath.os_path, OS_MODE_READ);
        if (keyfile == NULL)
        {
            filepath_set(&keypath, "keys.txt");
            keyfile = os_fopen(keypath.os_path, OS_MODE_READ);
        }
        if (keyfile == NULL)
        {
            filepath_set(&keypath, "keys.ini");
            keyfile = os_fopen(keypath.os_path, OS_MODE_READ);
        }
        if (keyfile == NULL)
        {
            filepath_set(&keypath, "prod.keys");
            keyfile = os_fopen(keypath.os_path, OS_MODE_READ);
        }
        if (keyfile == NULL)
        {
            /* Use $HOME/.switch/prod.keys if it exists */
            char *home = getenv("HOME");
            if (home == NULL)
                home = getenv("USERPROFILE");
            if (home != NULL)
            {
                filepath_set(&keypath, home);
                filepath_append(&keypath, ".switch");
                filepath_append(&keypath, "prod.keys");
                keyfile = os_fopen(keypath.os_path, OS_MODE_READ);
            }
        }
    }
    else if (keypath.valid == VALIDITY_VALID)
        keyfile = os_fopen(keypath.os_path, OS_MODE_READ);

    // Try to populate keyfile.
    if (keyfile != NULL)
    {
        printf("Loading '%s' keyset file\n", keypath.char_path);
        extkeys_initialize_keyset(&settings.keyset, keyfile);
        pki_derive_keys(&settings.keyset);
        fclose(keyfile);
    }
    else
    {
        printf("\n");
        fprintf(stderr, "Error: Unable to open keyset file\n"
                        "Use -k or --keyset to specify your keyset file path or place your keyset in ." OS_PATH_SEPARATOR "keys.dat\n");
        return EXIT_FAILURE;
    }

    // Make sure that key_area_key_application_keygen exists
    uint8_t has_keygen_key = 0;
    for (unsigned int i = 0; i < 0x10; i++)
    {
        if (settings.keyset.key_area_keys[settings.keygeneration - 1][0][i] != 0)
        {
            has_keygen_key = 1;
            break;
        }
    }
    if (has_keygen_key == 0)
    {
        fprintf(stderr, "Error: key_area_key_application for keygeneration %i is not present in keyset file\n", settings.keygeneration);
        return EXIT_FAILURE;
    }

    // Make sure that header_key exists
    uint8_t has_header_Key = 0;
    for (unsigned int i = 0; i < 0x10; i++)
    {
        if (settings.keyset.header_key[i] != 0)
        {
            has_header_Key = 1;
            break;
        }
    }
    if (has_header_Key == 0)
    {
        fprintf(stderr, "Error: header_key is not present in keyset file\n");
        return EXIT_FAILURE;
    }

    // Process NPDM
    printf("\n");
    printf("----> Processing NPDM\n");
    npdm_process(&settings, &cnmt_ctx);
    printf("\n");

    // Process NACP
    printf("----> Processing NACP\n");
    nacp_process(&settings);
    printf("\n");

    // Create NCAs
    nca_create_program(&settings, &cnmt_ctx);
    printf("\n");
    nca_create_control(&settings, &cnmt_ctx);
    printf("\n");
    if (settings.htmldoc_romfs_dir.valid == VALIDITY_VALID)
    {
        nca_create_manual_htmldoc(&settings, &cnmt_ctx);
        printf("\n");
    }
    if (settings.legalinfo_romfs_dir.valid == VALIDITY_VALID)
    {
        nca_create_manual_legalinfo(&settings, &cnmt_ctx);
        printf("\n");
    }
    nca_create_meta(&settings, &cnmt_ctx);
    printf("\n");

    // Create NSP
    printf("----> Creating NSP:\n");
    filepath_t nsp_file_path;
    filepath_init(&nsp_file_path);
    filepath_copy(&nsp_file_path, &settings.nsp_dir);
    filepath_append(&nsp_file_path, "%016" PRIx64 ".nsp", cnmt_ctx.cnmt_header.title_id);
    uint64_t pfs0_size;
    pfs0_build(&settings.nca_dir, &nsp_file_path, &pfs0_size);
    printf("\n----> Created NSP: %s\n", nsp_file_path.char_path);

    // Remove temp and nca directories
    printf("\n");
    if (settings.keepncadir == 1)
    {
        printf("Removing created temp directory\n");
        filepath_remove_directory(&settings.temp_dir);
    }
    else
    {
        printf("Removing created temp and nca directories\n");
        filepath_remove_directory(&settings.temp_dir);
        filepath_remove_directory(&settings.nca_dir);
    }

    // Summary
    printf("\n\n");
    printf("Summary:\n\n");
    printf("Title ID: %016" PRIx64 "\n", cnmt_ctx.cnmt_header.title_id);
    printf("SDK Version: %" PRId8 ".%" PRId8 ".%" PRId8 ".%" PRId8 "\n", settings.sdk_major, settings.sdk_minor, settings.sdk_micro, settings.sdk_revision);
    if (settings.plaintext == 0)
        printf("Section Crypto Type: Regular Crypto\n");
    else
        printf("Sections Crypto Type: Plaintext\n");
    printf("Keygeneration: %i\n", settings.keygeneration);
    char keyareakey_hex[33];
    hexBinaryString(settings.keyareakey, 16, keyareakey_hex, 33);
    keyareakey_hex[32] = '\0';
    printf("Key area key 2: %s\n", keyareakey_hex);
    if (settings.noromfs == 0)
        printf("Program NCA RomFS Section: Yes\n");
    else
        printf("Program NCA RomFS Section: No\n");
    if (settings.nologo == 0)
        printf("Program NCA Logo Section: Yes\n");
    else
        printf("Program NCA Logo Section: No\n");
    if (settings.htmldoc_romfs_dir.valid == VALIDITY_VALID)
        printf("HtmlDoc NCA: Yes\n");
    else
        printf("HtmlDoc NCA: No\n");
    if (settings.legalinfo_romfs_dir.valid == VALIDITY_VALID)
        printf("LegalInfo NCA: Yes\n");
    else
        printf("LegalInfo NCA: No\n");
    printf("Created NSP: %s\n", nsp_file_path.char_path);

    free(settings.keyareakey);
    return EXIT_SUCCESS;
}