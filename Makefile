all: test

# Overridable so cross-compilation (e.g. multi-arch container builds) can target
# another platform. GOARM is inherited from the environment when set.
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0

ENVVAR = GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED)
GOLANGCI_INSTALLED := $(shell which bin/golangci-lint)


.PHONY: all build container clean gofmt lint test

lint:
ifdef GOLANGCI_INSTALLED
	bin/golangci-lint run -E revive -E goimports
else
	@echo Warning golangci-lint not installed. Skipping linting
	@echo Installation instructions: https://github.com/golangci/golangci-lint#ci-installation
endif

build: clean gofmt
	$(ENVVAR) go build -o kube-monkey

docker_args=
ifdef http_proxy
docker_args+= --build-arg http_proxy=$(http_proxy)
endif
ifdef https_proxy
docker_args+= --build-arg https_proxy=$(https_proxy)
endif

# Suppressing docker build avoids printing the env variables
container:
	@echo "Running docker with '$(docker_args)'"
	@docker build $(docker_args) -t kube-monkey:latest .

gofmt:
	gofmt -s -w .

# Same as gofmt, but also orders imports
goimports:
	goimports -w .

clean:
	rm -f kube-monkey

test: build
	go test -v -cover -gcflags=-l ./...
