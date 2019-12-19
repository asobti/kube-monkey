<<<<<<< HEAD
########################
### Builder          ###
########################
FROM golang:latest as builder

RUN curl https://glide.sh/get | sh
COPY . /go/src/github.com/asobti/kube-monkey
WORKDIR /go/src/github.com/asobti/kube-monkey
RUN glide install
RUN make build

########################
### Final            ###
########################
FROM scratch
COPY --from=builder /go/src/github.com/asobti/kube-monkey/kube-monkey /go/bin/kube-monkey
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

ENTRYPOINT ["/go/bin/kube-monkey"]
=======
FROM ubuntu:19.04
RUN if (dpkg -l | grep -cq tzdata); then \
        echo "tzdata package already installed! Skipping tzdata installation"; \
    else \
        echo "Installing tzdata to avoid go panic caused by missing timezone data"; \
        apt-get update && apt-get install -y --no-install-recommends \
		apt-utils \
		tzdata \
	&& rm -rf /var/lib/apt/lists/*; \
    fi
COPY kube-monkey /kube-monkey
>>>>>>> upstream/master
