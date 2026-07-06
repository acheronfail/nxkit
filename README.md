# NXKit

An application intended to bring some commonly needed tools for Nintendo Switch Homebrew to a more user-friendly place. Currently supports:

- 🐧️ Linux, 🍎️ macOS and 🪟️ Windows!
- 💉️ injecting payloads
- 🚀️ creating custom launchers
- 🔎️ exploring NAND dumps
- 🔪️ splitting/merging NSPs
- 🗜️ compressing/decompressing NSZs

Hopefully it's helpful to you!

## Installation

Head over to the [releases page](https://github.com/acheronfail/nxkit/releases) and download the latest release for your platform.

If you're using macOS, be aware that this application isn't notarised (since that requires paying Apple just to sign an app, and this is free so it's not happening). This means that you may need to run the following command in your terminal to allow the app to run:

```bash
xattr -d com.apple.quarantine /path/to/NXKit.app
```
This command removes the quarantine attribute from the app, allowing it to run without being blocked by macOS's Gatekeeper.
