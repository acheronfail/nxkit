# Development

- [Development](#development)
  - [Dependencies](#dependencies)
    - [Linux](#linux)
    - [macOS](#macos)
    - [Windows](#windows)
  - [Project setup](#project-setup)


## Dependencies

You need:

- NodeJS (I recommend using [`fnm`](https://github.com/Schniz/fnm))
- Python 3.11 or higher
- OpenSSL
- [`just`](https://github.com/casey/just)

Installation of the above depends on your platform. Some guidelines are provided below, but really it's up to you how you want to install them since you're the administrator of your machine.

### Linux

```shell
# debian based
apt install python3 libssl-dev
```

```shell
# arch based
pacman -S python openssl
```

### macOS

```shell
brew install python openssl
```

### Windows

Here's where it gets tricky. Once you've installed NodeJS and have that setup, then I recommend installing OpenSSL via [`vcpkg`](https://github.com/microsoft/vcpkg). Something like this should work:

```powershell
# install vcpkg
git clone https://github.com/microsoft/vcpkg.git
cd vcpkg; .\bootstrap-vcpkg.bat

# install openssl
.\vcpkg.exe install openssl:x64-windows-static
```

Now you've got OpenSSL installed, you'll need to set these two environment variables so that the native node modules in this repository can use it:

```powershell
$env:OPENSSL_LIB_DIR = "C:\path\to\vcpkg\installed\x64-windows-static\lib"
$env:OPENSSL_INC_DIR = "C:\path\to\vcpkg\installed\x64-windows-static\include"
```

Now, with those environment variables set you should be able to build and run the application.

## Project setup

Everything is done via [`just`](https://github.com/casey/just).
Run `just` by itself to see the available commands. But really, you should just need:

```shell
# installs node dependencies and builds native node modules
just setup

# run this to open the app in dev mode
just dev
```
