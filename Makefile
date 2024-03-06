COMMIT ?= $$(git rev-parse HEAD)
TAG ?= $$(git describe --tags --abbrev=0 2>/dev/null || echo dev)

init:
	go mod init github.com/psvmcc/disconter

deps:
	go mod tidy
	go mod vendor

pre-test:
	go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

test: pre-test
	go vet -mod=vendor $(shell go list ./...)
	go vet -mod=vendor -vettool=$(shell which shadow) $(shell go list ./...)
	golangci-lint run main.go

clean:
	rm -rf build
	mkdir build

.PHONY: build
build: clean
	go build -mod=vendor -ldflags "-X main.version=${TAG} -X main.commit=${COMMIT}" -o build/disconter main.go

linux:
	env GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags "-X main.version=${TAG} -X main.commit=${COMMIT}" -o build/disconter.linux main.go

container:
	podman build -t psvmcc/disconter .

run:
	env DOCKER_SOCKET=/Users/ps/.local/share/containers/podman/machine/qemu/podman.sock go run main.go
