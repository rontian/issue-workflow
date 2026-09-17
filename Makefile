.PHONY: test vet race build install

test:
	go test ./...

vet:
	go vet ./...

race:
	go test -race ./...

build:
	go build ./cmd/iw

install:
	go install ./cmd/iw
