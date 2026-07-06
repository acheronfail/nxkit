# Need help sending a payload?

Usually on Linux things should just work.

But, if you're having trouble ensure your user is part of the `plugdev` group (`sudo usermod -aG plugdev $USER`) and add the following `udev` rule:

```
# /etc/udev/rules.d/50-switch.rules
SUBSYSTEM=="usb", ATTR{idVendor}=="0955", MODE="0664", GROUP="plugdev"
```

Visit [sending_payload](https://nh-server.github.io/switch-guide/user_guide/sysnand/sending_payload/) for more information about sending payloads.
