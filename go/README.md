This directory is contains a Work-In-Progress Go implementation of NXKit.

It's not complete, but it's a start - mainly to see if leveraging Go's simple cross-platform outputs makes things a lot simpler than dealing with electron, native node modules and wasm.

## Credits

Could not have written the FAT driver without:

- [the wonderful FatFs](https://elm-chan.org/fsw/ff/)
- [go-diskfs](https://github.com/diskfs/go-diskfs)

And also the XTSN code is ported from my NodeJS implementation, which I couldn't have done without:

- <https://github.com/DacoTaco/YASDU/blob/master/R.N.D/xts_crypto.cpp>
- <https://github.com/luigoalma/haccrypto/blob/master/haccrypto/_crypto.cpp>
- <https://github.com/eliboa/NxNandManager/blob/master/NxNandManager/NxCrypto.cpp>
- <https://github.com/ihaveamac/ninfs/blob/main/ninfs/mount/nandhac.py>
- <https://gitlab.com/roothorick/busehac>
- <https://github.com/ihaveamac/switchfs>
