# Image configurations
ARG GOLANG_VERSION
FROM golang:{GOLANG_VERSION:-1.10} as builder

COPY . $GOPATH/src/github.com/asobti/kube-monkey
WORKDIR $GOPATH/src/github.com/asobti/kube-monkey
RUN make build

FROM ubuntu
LABEL maintainer="asobti <ayushsobti@gmail.com>"

RUN if (dpkg -l | grep -cq tzdata); then \
        echo "tzdata package already installed! Skipping tzdata installation"; \
    else \
        echo "Installing tzdata to avoid go panic caused by missing timezone data"; \
        apt-get update && apt-get install -y --no-install-recommends \
		apt-utils \
		tzdata \
	&& rm -rf /var/lib/apt/lists/*; \
fi

COPY --from=builder /go/src/github.com/asobti/kube-monkey/kube-monkey /kube-monkey
