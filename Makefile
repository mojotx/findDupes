CC      ?= cc
CFLAGS  ?= -O2 -Wall -Wextra -pedantic -std=c11
LDFLAGS ?=

# macOS CommonCrypto is a system framework – no extra link flags needed.

TARGET  := findDupes
SRC     := main.c

.PHONY: all clean

all: $(TARGET)

$(TARGET): $(SRC)
	$(CC) $(CFLAGS) $(LDFLAGS) -o $@ $<

clean:
	rm -f $(TARGET)
