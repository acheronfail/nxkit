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
      ],
      "conditions": [
        ["OS=='mac'", {
          "libraries": [
            "<!(sh -c 'echo -L$(pkg-config --variable=libdir openssl)/lib -lssl -lcrypto')"
          ]
        }],
      ]
    }
  ]
}
