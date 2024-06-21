# Entering RCM

As the Switch uses a Tegra X1 processor, it has a special recovery mode that is, in most scenarios, useless for the end-user. Fortunately, due to the fusee-gelee vulnerability, this special mode acts as our gateway into CFW.

Go to [sending_payload](https://nh-server.github.io/switch-guide/user_guide/sysnand/sending_payload/) for more information.

- TODO: description of how to enter RCM mode
- TODO: doc linux udev: `SUBSYSTEM=="usb", ATTR{idVendor}=="0955", MODE="0664", GROUP="plugdev"` @ `/etc/udev/rules.d/50-switch.rules`
- TODO: doc windows usb driver install:
  - download https://zadig.akeo.ie/
  - connect Switch in RCM
  - choose `APX`
  - select `libusbK`
  - select `Install Driver`
