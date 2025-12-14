CC = gcc
CFLAGS = -I../../../../vendor/hacbrewpack
SRCS = main.c \
       ../../../../vendor/hacbrewpack/romfs.c \
       ../../../../vendor/hacbrewpack/filepath.c \
       ../../../../vendor/hacbrewpack/utils.c \
       ../../../../vendor/hacbrewpack/ConvertUTF.c
TARGET = romfs_gen

all: $(TARGET)

$(TARGET): $(SRCS)
	$(CC) $(CFLAGS) $(SRCS) -o $(TARGET)

clean:
	rm -f $(TARGET)
