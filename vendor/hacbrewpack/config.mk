# from hacBrewPack's template
CC = gcc
CFLAGS = -Wall -Wextra -pedantic -std=gnu11 -fPIC
LDFLAGS = -lmbedtls -lmbedx509 -lmbedcrypto

# options for enscripten
# https://emsettings.surma.technology/
CC = emcc
CFLAGS += -flto -Os
LDFLAGS += -s ENVIRONMENT='web,webview,worker,node'
LDFLAGS += -s MODULARIZE=1 -s "EXPORT_NAME='hacbrewpack'"
LDFLAGS += -s EXPORTED_RUNTIME_METHODS='callMain,FS'
LDFLAGS += -s EXPORT_ES6=1
LDFLAGS += -s TOTAL_STACK=512mb
LDFLAGS += -s ALLOW_MEMORY_GROWTH=1
LDFLAGS += -s EXIT_RUNTIME=1
