all: test

ENVVAR = GOOS=linux GOARCH=amd64 CGO_ENABLED=0
TAG = v0.2.3
GOLANGCI_INSTALLED := $(shell which bin/golangci-lint 2>/dev/null)


.PHONY: all build containers alpine ubuntu clean gofmt lint test

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
proxies:
ifneq ($(and $(http_proxy), $(https_proxy)),)
	@echo Starting Docker build, importing both http_proxy and https_proxy env variables
dockerbuild=docker build --build-arg http_proxy=$(http_proxy) --build-arg https_proxy=$(https_proxy)
else
ifdef http_proxy
	@echo Starting Docker build, importing http_proxy
dockerbuild= docker build --build-arg http_proxy=$(http_proxy)
else
ifdef https_proxy
	@echo Starting Docker build, importing https_proxy
dockerbuild= docker build --build-arg https_proxy=$(https_proxy)
else
	@echo no env proxies set, building normally
dockerbuild= docker build 
endif
endif
endif

# Supressing docker build avoids printing the env variables
containers: proxies alpine ubuntu test
	@echo Building all containers

alpine:
	@echo $(dockerbuild) -t kube-monkey:$(TAG) alpine
	@$(dockerbuild) -t kube-monkey:$(TAG)_alpine alpine
ubuntu:
	@$(dockerbuild) -t kube-monkey:$(TAG)_ubuntu ubuntu

gofmt:
	@echo Checking gofmt:
	find . -path ./vendor -prune -o -name '*.go' -print | xargs -L 1 -I % gofmt -s -w %

clean:
	@echo Cleaning compiled kube-monkey binary:
	rm -f kube-monkey

test: build
	go test -v -cover ./...
