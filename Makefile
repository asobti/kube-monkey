all: test

ENVVAR = GOOS=linux GOARCH=amd64 CGO_ENABLED=0
TAG = v0.2.3
GOLANGCI_RELEASE = v1.4.1

.PHONY: all build container clean gofmt lint test

bootstrap:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s $(GOLANGCI_RELEASE)

lint:
	bin/golangci-lint run -E golint -E goimports

build: clean gofmt lint
	$(ENVVAR) go build -o kube-monkey

# Supressing docker build avoids printing the env variables
container: test
ifneq ($(and $(http_proxy), $(https_proxy)),)
	@echo Starting Docker build, importing both http_proxy and https_proxy env variables
	@docker build --build-arg http_proxy=$(http_proxy) --build-arg https_proxy=$(https_proxy) -t kube-monkey:$(TAG) .
else
ifdef http_proxy
	@echo Starting Docker build, importing http_proxy
	@docker build --build-arg http_proxy=$(http_proxy) -t kube-monkey:$(TAG) .
else
ifdef https_proxy
	@echo Starting Docker build, importing https_proxy
	@docker build --build-arg https_proxy=$(https_proxy) -t kube-monkey:$(TAG) .
else
	@echo no env proxies set, building normally
	docker build -t kube-monkey:$(TAG) .
endif
endif
endif

gofmt:
	find . -path ./vendor -prune -o -name '*.go' -print | xargs -L 1 -I % gofmt -s -w %

clean:
	rm -f kube-monkey

test: build
	go test -v -cover ./...
