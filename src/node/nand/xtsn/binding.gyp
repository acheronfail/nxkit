{
  "targets": [
    {
      "target_name": "xtsn",
      "sources": [
        "native.cc"
      ],
      "include_dirs": [
        "<!(pkg-config --cflags-only-I openssl | sed 's/-I//g')"
      ],
      "libraries": [
        "<!(pkg-config --libs openssl)"
      ]
    }
  ]
}
