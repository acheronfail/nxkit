{
  "targets": [
    {
      "target_name": "xtsn",
      "sources": [ "xtsn.cpp" ],
      "include_dirs": [],
      "conditions": [
        [ "OS=='mac'", {
          "libraries": [
            "<!(sh -c 'echo -L$(pkg-config --variable=libdir openssl) -lssl -lcrypto')"
          ]
        }],
        [ "OS=='linux'", {
          "libraries": [ "-lssl", "-lcrypto" ]
        }],
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
        }]
      ]
    }
  ]
}
