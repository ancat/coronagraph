.PHONY: all build clean lint vet fmt fmt-check tidy check

BINARY := cg

all: build

build:
	go build -o $(BINARY) .

# Run go vet (built-in static analysis)
vet:
	go vet ./...

# Format all source files in place
fmt:
	gofmt -w .

# Check formatting without modifying files (CI-friendly)
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "The following files are not gofmt'd:"; gofmt -l .; exit 1)

# Tidy and verify the module graph
tidy:
	go mod tidy
	go mod verify

# Full CI check: format, vet, lint, build
check: fmt-check vet lint build

clean:
	rm -f $(BINARY)
