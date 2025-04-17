{
  "targets": [
    {
      "target_name": "xtsn",
      "sources": [
        "xtsn.cpp"
      ],
      "include_dirs": [
        "<!(pkg-config --cflags-only-I openssl | sed 's/-I//g')"
      ],
      "libraries": [
        "<!(pkg-config --libs openssl)"
      ],
      "conditions": [
        [ "OS=='win'", {
          "libraries": [
            "<!(echo %OPENSSL_LIB_DIR%\\libssl.lib)",
            "<!(echo %OPENSSL_LIB_DIR%\\libcrypto.lib)"
          ],
          "msvs_settings": {
            "VCLinkerTool": {
              "AdditionalDependencies": [
                "libssl.lib",
                "libcrypto.lib"
              ],
              "AdditionalLibraryDirectories": [
                "<!(echo %OPENSSL_LIB_DIR%)"
              ]
            },
            "VCCLCompilerTool": {
              "AdditionalIncludeDirectories": [
                "<!(echo %OPENSSL_LIB_DIR%)"
              ]
            }
          }
        }],
        ["OS=='mac'", {
          "libraries": [
            "<!(sh -c 'echo -L$(pkg-config --variable=libdir openssl)/lib -lssl -lcrypto')"
          ]
        }],
      ]
    }
  ]
}
