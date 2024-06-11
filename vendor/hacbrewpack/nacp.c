#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <sys/time.h>
#include "nacp.h"
#include "filepath.h"

void nacp_process(hbp_settings_t *settings)
{
    filepath_t nacp_filepath;
    filepath_init(&nacp_filepath);
    filepath_copy(&nacp_filepath, &settings->control_romfs_dir);
    filepath_append(&nacp_filepath, "control.nacp");

    FILE *fl;
    fl = os_fopen(nacp_filepath.os_path, OS_MODE_EDIT);
    if (fl == NULL)
    {
        fprintf(stderr, "Failed to open %s!\n", nacp_filepath.char_path);
        exit(EXIT_FAILURE);
    }

    // Read NACP
    nacp_t nacp;
    memset(&nacp, 0, sizeof(nacp));
    if (fread(&nacp, 1, sizeof(nacp_t), fl) != sizeof(nacp_t))
    {
        fprintf(stderr, "Failed to read control.nacp!\n");
        exit(EXIT_FAILURE);
    }

    char tname = 0x00;
    char tpub = 0x00;
    for (int i = 0; i <= 15; i++)
    {
        tname = nacp.Title[i].Name[0];
        tpub = nacp.Title[i].Publisher[0];
        if (tname != 0x00 && tpub != 0x00)
            break;
    }

    // Override title name if specified
    if (settings->titlename[0] != 0x00)
    {
        printf("Changing Title Name\n");
        for (int j = 0; j <= 11; j++)
        {
            memset(nacp.Title[j].Name, 0, 0x200);
            strcpy(nacp.Title[j].Name, settings->titlename);
        }
    }
    else
    {
        // Check Title Name
        printf("Validating Title Name\n");
        if (tname == 0)
        {
            fprintf(stderr, "Error: Invalid Title Name in control.nacp\n");
            exit(EXIT_FAILURE);
        }
    }

    // Override title publisher if specified
    if (settings->titlepublisher[0] != 0x00)
    {
        printf("Changing Title Publisher\n");
        for (int k = 0; k <= 11; k++)
        {
            memset(nacp.Title[k].Publisher, 0, 0x100);
            strcpy(nacp.Title[k].Publisher, settings->titlepublisher);
        }
    }
    else
    {
        // Check Publisher
        printf("Validating Title Publisher\n");
        if (tpub == 0)
        {
            fprintf(stderr, "Error: Invalid Publisher in control.nacp\n");
            exit(EXIT_FAILURE);
        }
    }

    // Change logo handeling to Auto
    if (settings->nopatchnacplogo == 0)
    {
        printf("Changing logo handeling to auto\n");
        nacp.LogoHandling = 0x00;
    }

    if (settings->title_id != 0)
    {
        printf("Setting TitleIDs\n");
        nacp.PresenceGroupId = settings->title_id;
        nacp.SaveDataOwnerId = settings->title_id;
        nacp.AddOnContentBaseId = settings->title_id + 0x1000;
        for (int x = 0; x < 8; x++)
            nacp.LocalCommunicationId[x] = settings->title_id;
    }

    // Backup and re-write NACP
    if (settings->titlename[0] != 0x00 || settings->titlepublisher[0] != 0x00 || settings->title_id != 0 || settings->nopatchnacplogo == 0)
    {
        // Copy control.nacp to backup directory
        struct timeval ct;
        gettimeofday(&ct, NULL);
        filepath_t nacp_filepath;
        filepath_t bkup_nacp_filepath;
        filepath_init(&nacp_filepath);
        filepath_copy(&nacp_filepath, &settings->control_romfs_dir);
        filepath_append(&nacp_filepath, "control.nacp");
        filepath_init(&bkup_nacp_filepath);
        filepath_copy(&bkup_nacp_filepath, &settings->backup_dir);
        filepath_append(&bkup_nacp_filepath, "%" PRIu64 "_control.nacp", ct.tv_sec);
        printf("Backing up control.nacp\n");
        filepath_copy_file(&nacp_filepath, &bkup_nacp_filepath);
        printf("Writing control.nacp\n");
        fseeko64(fl, 0, SEEK_SET);
        fwrite(&nacp, 1, sizeof(nacp_t), fl);
    }

    fclose(fl);
}