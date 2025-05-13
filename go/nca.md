```py
i# Common pattern for all NCA types (Program, Control, Manual, Meta):
def create_nca(type):
    # 1. Initialize empty NCA header
    header = create_empty_nca_header()
    
    # 2. Create NCA file and write placeholder header
    nca_file = create_file("temp.nca")
    write_placeholder_header(nca_file)

    # 3. Build and write sections based on NCA type
    if type == "Program":
        # Section 0: ExeFS (executable content)
        build_exefs_section()
        write_exefs_to_nca()
        
        # Section 1: RomFS (game data)
        if has_romfs:
            build_romfs_section() 
            write_romfs_to_nca()
            
        # Section 2: Logo
        if has_logo:
            build_logo_section()
            write_logo_to_nca()

    elif type == "Control":
        # Section 0: RomFS (control data/metadata)
        build_control_romfs()
        write_romfs_to_nca()

    elif type == "Manual":
        # Section 0: RomFS (manual/legal docs)
        build_manual_romfs()
        write_romfs_to_nca()

    elif type == "Meta":
        # Section 0: PFS0 (CNMT metadata)
        build_cnmt_section()
        write_cnmt_to_nca()

    # 4. Update header with section info
    update_section_offsets()
    calculate_section_hashes()
    
    # 5. Finalize NCA
    if not plaintext:
        encrypt_sections()
    encrypt_key_area()
    encrypt_header()
    write_final_header()

    # 6. Post-processing
    calculate_nca_hash()
    rename_to_final_name()

# Main flow:
create_nca("Program")    # Game executable and data
create_nca("Control")    # Game metadata/icon
create_nca("Manual")     # Optional manual/legal docs
create_nca("Meta")       # Package metadata (CNMT)
```
