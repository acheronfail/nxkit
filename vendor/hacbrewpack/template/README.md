# hacBrewPack - Template

## Makefile

Generic 'Makefile' that you can use to compile homebrews  

## npdm.json

Generic NPDM json file which libnx reads during compile and makes npdm from it  
It needs to be modified based on your app  
Check switchbrew for more information: [http://switchbrew.org/index.php/NPDM](http://switchbrew.org/index.php/NPDM)

## Folder structure

Default folders structure:
```
hacbrewpack
|   hackbrewpack(.exe)
|   keys.dat
|
|___exefs
|   |   main
|   |   main.npdm
|
|___romfs
|
|___logo
|   |   NintendoLogo.png
|   |   StartupMovie.gif
|
|___control
|   |   control.nacp
|   |   icon_AmericanEnglish.dat
```

You can skip creating program romfs with --noromfs and creating program logo with --nologo  
You can use options to change default folders paths  
  
Default output folders are:  
hacbrewpack_out: contains nsp  
hacbrewpack_nca: contains ncas  
hacbrewpack_temp: temporary files  

## Keygeneration

Keygeneration is a key that hacBrewPack use to encrypt key area in ncas  
It is a number between 1-32 and it describes the 'key_area_key_application' key that hacBrewPack use in encryption  
Firmwares always support applications with keygenerations up to the keygeneration they ship with  

Keygeneration | Firmware
--------------| --------
1 | 1.0.0 - 2.3.0
2 | 3.0.0
3 | 3.0.1 - 3.0.2
4 | 4.0.0 - 4.1.0
5 | 5.0.0 - 5.1.0
6 | 6.0.0 - 6.1.0
7 | 6.2.0
8 | 7.0.0 - 8.0.1
9 | 8.1.0
