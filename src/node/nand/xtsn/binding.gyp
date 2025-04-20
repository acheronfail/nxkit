{
  "targets": [
    {
      "target_name": "xtsn",
      "sources": [ "xtsn.cpp" ],
      "include_dirs": [],
      "conditions": [
        [ "OS=='mac'", {
          "include_dirs": [
            "<!(pkg-config --variable=includedir openssl)"
          ],
          "libraries": [
            "<!(sh -c 'echo $(pkg-config --variable=libdir openssl)/libssl.a')",
            "<!(sh -c 'echo $(pkg-config --variable=libdir openssl)/libcrypto.a')"
          ]
        }],
        [ "OS=='linux'", {
          "libraries": [ "-l:libssl.a", "-l:libcrypto.a" ]
        }],
        [ "OS=='win'", {
          "include_dirs": [
            "<!(echo %OPENSSL_INC_DIR%)"
          ],
          "libraries": [
            "<!(echo %OPENSSL_LIB_DIR%\\libssl.lib)",
            "<!(echo %OPENSSL_LIB_DIR%\\libcrypto.lib)",
            "crypt32.lib",
            "ws2_32.lib",
            "user32.lib"
          ],
          "msvs_settings": {
            "VCLinkerTool": {
              "AdditionalDependencies": [
                "libssl.lib",
                "libcrypto.lib",
                "crypt32.lib",
                "ws2_32.lib",
                "user32.lib"
              ],
              "AdditionalLibraryDirectories": [
                "<!(echo %OPENSSL_LIB_DIR%)"
              ]
            },
            "VCCLCompilerTool": {
              "RuntimeLibrary": 0,
              "AdditionalIncludeDirectories": [
                "<!(echo %OPENSSL_LIB_DIR%)",
                "<!(echo %OPENSSL_INC_DIR%)"
              ]
            }
          },
          "defines": [
            "OPENSSL_NO_DYNAMIC_ENGINE"
          ]
        }]
      ]
    }
  ]
}
