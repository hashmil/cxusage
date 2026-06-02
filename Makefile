.PHONY: test vet build install clean

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o bin/cxusage ./cmd/cxusage

install:
	go install ./cmd/cxusage

clean:
	rm -rf bin dist coverage.out
