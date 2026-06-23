.PHONY: all build clean

BINARY := cg

all: build

build:
	go build -o $(BINARY) .

clean:
	rm -f $(BINARY)
