all: test

ENVVAR = GOOS=linux GOARCH=amd64 CGO_ENABLED=0
TAG := $(shell cat VERSION)
GOLANGCI_INSTALLED := $(shell which bin/golangci-lint 2>/dev/null)


.PHONY: all build containers alpine ubuntu ubuntu-compat clean gofmt lint test

lint:
ifdef GOLANGCI_INSTALLED
	bin/golangci-lint run -E golint -E goimports
else
	@echo Warning golangci-lint not installed. Skipping linting
	@echo Installation instructions: https://github.com/golangci/golangci-lint #ci-installation
endif

build: clean gofmt lint
	$(ENVVAR) go build -o kube-monkey

# If you are running behind a proxy with self signed certs,
# you will have to change the Dockerfiles 
#-RUN go get github.com/asobti/kube-monkey
#+ENV GIT_SSL_NO_VERIFY=1
#+RUN go get -v -insecure github.com/asobti/kube-monkey
docker_args=
ifdef http_proxy
docker_args+= --build-arg http_proxy=$(http_proxy)
endif
ifdef https_proxy
docker_args+= --build-arg https_proxy=$(https_proxy)
endif

# Supressing docker build avoids printing the env variables
containers: test alpine ubuntu
	@echo "Building all containers with '$(docker_args)'"

alpine:
	docker build $(docker_args) -t kube-monkey:$(TAG) alpine
ubuntu:
	docker build $(docker_args) -t kube-monkey:$(TAG)_ubuntu ubuntu

# Docker compatibility mode support
# If running Docker version < 17, multi-stage builds are not supported
# this target uses a single-stage build to make the container
ubuntu-compat: clean build
	docker build $(docker_args) -t kube-monkey:$(TAG)_ubuntu -f ubuntu_compat/Dockerfile .

gofmt:
	@echo Checking gofmt:
	find . -path ./vendor -prune -o -name '*.go' -print | xargs -L 1 -I % gofmt -s -w %

clean:
	@echo Cleaning compiled kube-monkey binary:
	rm -f kube-monkey

test: build
	go test -v -cover ./...
