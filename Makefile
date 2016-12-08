all: build

ENVVAR = GOOS=linux GOARCH=amd64 CGO_ENABLED=0
TAG = v0.1.0

build: clean
	$(ENVVAR) go build -o kube-monkey

container: build
	docker build -t kube-monkey:$(TAG) .

clean:
	rm -f kube-monkey

.PHONY: all build container clean
