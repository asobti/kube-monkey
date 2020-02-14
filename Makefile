all: test

ENVVAR = GOOS=linux GOARCH=amd64 CGO_ENABLED=0
TAG := $(shell cat VERSION)
GOLANGCI_INSTALLED := $(shell which bin/golangci-lint)


.PHONY: all build container clean gofmt lint test

# linting is temporarily disabled
# see https://github.com/asobti/kube-monkey/pull/123
lint:
ifdef GOLANGCI_INSTALLED
	bin/golangci-lint run -E golint -E goimports
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

# Supressing docker build avoids printing the env variables
container:
	@echo "Running docker with '$(docker_args)'"
	@docker build $(docker_args) -t kube-monkey:$(TAG) .

gofmt:
	find . -path ./vendor -prune -o -name '*.go' -print | xargs -L 1 -I % gofmt -s -w %

# Same as gofmt, but also orders imports
goimports:
	find . -path ./vendor -prune -o -name '*.go' -print | xargs -L 1 -I % goimports -w %

clean:
	rm -f kube-monkey

test: build
	go test -gcflags=-l -v -cover ./...
