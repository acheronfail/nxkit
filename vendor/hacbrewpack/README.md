# hacBrewPack

[![License: GPL v2](https://img.shields.io/badge/License-GPL%20v2-blue.svg)](https://www.gnu.org/licenses/old-licenses/gpl-2.0.en.html)

hacBrewPack is a tool for creating Nintendo Switch NCAs (Nintendo Content Archive) from homebrews and pack them into NSPs (Nintendo Submission Package)  
  
Thanks: SciresM, yellows8, Adubbz, SwitchBrew

## Usage

### Keys

You should place your keyset file with "keys.dat" filename in the same folder as hacBrewPack  
hacBrewPack tries to load "keys.txt", "keys.ini", "prod.keys" and "~/.switch/prod.keys" files if "keys.dat" doesn't exist  
Alternatively, You can use -k or --keyset option to load your keyset file  
Required keys are:  

Key Name | Description
-------- | -----------
header_key | NCA Header Key
key_area_key_application_xx | Application key area encryption keys

### Compiling Homebrew

You need to compile homebrew with proper Makefile, you can use the one in template folder  
You must use valid lower-case titleid in Makefile and npdm.json. Valid titleid range is: 0x0100000000000000 - 0x0fffffffffffffff  
It's not suggested to use a titleid higher than 0x01ffffffffffffff  
Both titleids in Makefile and npdm.json must be the same  
Compiled homebrew must have following files:  

```
build\exefs\main  
build\exefs\main.npdm  
[TARGET].nacp  
```

You must place created 'main' and 'main.npdm' files in exefs folder, you can find them in build/exefs  
You must rename created nacp file to 'control.nacp' and place it in control folder  

### Icon

You should place your icon with "icon_{Language}.dat" file name in control folder, "icon_AmericanEnglish.dat" is the default one if you don't manually edit your nacp. dat files are just renamed jpg files  
Check [switchbrew](http://switchbrew.org/index.php/Settings_services#LanguageCode) for more info about language names  
Your icon file format must be JPEG with 256x256 dimensions  
It's highly recommended to delete unnecessary exif data from your jpeg file (easy way: Open icon with GIMP or Paint, save as bmp, Open it again and save as jpg)  
If you see placeholder instead of icon after installing nsp, It's likely due to exif data  
If you have some exif data that horizon os doesn't like (like Camera Brand), Your app may leave in installing state in qlaunch  
If you don't put your icon in control folder, you'll see a general icon after installing nsp (I don't recommend this)  

### Logo

"logo" folder should contain "NintendoLogo.png" and "StartupMovie.gif". They'll appear when the app is loading  
Both files are not licensed according to [switchbrew](http://switchbrew.org/index.php/NCA_Content_FS) but i didn't include them anyway. You can also replace these files with custom ones  
You can use --nologo if you don't have any custom logos and you don't have the original ones, as the result switch will show a black screen without nintendo logo at top left and switch animation on bottom right  

### CLI Options

```
*nix: ./hacbrewpack [options...]  
Windows: .\hacbrewpack.exe [options...]  
  
Options:  
-k, --keyset             Set keyset filepath, default filepath is ./keys.dat  
-h, --help               Display usage  
--nspdir                 Set output nsp directory path, default path is ./hacbrewpack_nsp/  
--ncadir                 Set output nca directory path, default path is ./hacbrewpack_nca/  
--tempdir                Set temp directory filepath, default filepath is ./hacbrewpack_temp/  
--backupdir              Set output nsp directory path, default path is ./hacbrewpack_backup/  
--exefsdir               Set program exefs directory path, default path is ./exefs/  
--romfsdir               Set program romfs directory path, default path is ./romfs/  
--logodir                Set program logo directory path, default path is ./logo/  
--controldir             Set control romfs directory path, default path is ./control/  
--htmldocdir             Set HtmlDocument romfs directory path  
--legalinfodir           Set LegalInformation romfs directory path  
--noromfs                Skip creating program romfs section  
--nologo                 Skip creating program logo section  
--keygeneration          Set keygeneration for encrypting key area keys  
--keyareakey             Set Set key area key 2 in hex with 16 bytes lenght  
--sdkversion             Set SDK version in hex, default SDK version is 000C1100  
--plaintext              Skip encrypting sections and set section header block crypto type to plaintext  
--keepncadir             Keep NCA directory  
--nosignncasig2          Skip patching acid public key in npdm and signing nca header with acid public key  
Overriding options:  
--titleid                Use specified titleid for creating ncas and patch titleid in npdm and nacp  
--titlename              Change title name in nacp for all languages, max size is 512 bytes  
--titlepublisher         Change title publisher in nacp for all languages, max size is 256 bytes  
--nopatchnacplogo        Skip changing logo handeling to auto in NACP  
```

hacBrewPack doesn't need any options to work. if you follow folder structure properly, you can just run the program and it'll make a NSP  
Check template folder for default folder structure, Makefile, npdm json and other useful info  

## Licensing

This software is licensed under the terms of the GNU General Public License, version 2.  
You can find a copy of the license in the LICENSE file.  
Portions of project hacBrewPack are parts of other projects, make sure to check LICENSES folder