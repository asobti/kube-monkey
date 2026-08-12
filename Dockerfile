########################
### Builder          ###
########################
# Pinned to the build platform so the compiler always runs natively and
# cross-compiles to the target, instead of running the whole toolchain
# under QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder

# BuildKit supplies these per target platform. The defaults only apply to the
# classic builder, which leaves them unset.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT=

WORKDIR /kube-monkey

# Dependencies resolve in their own layer so source edits don't re-download them.
COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

ENV GOOS=$TARGETOS GOARCH=$TARGETARCH
# linux/arm/v7 maps to GOARM=7; every other target ignores it.
RUN if [ "$TARGETVARIANT" = "v7" ]; then export GOARM=7; fi; \
    make build

########################
### Final            ###
########################
FROM scratch
# Needed to verify TLS when posting to notification webhooks.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /kube-monkey/kube-monkey /kube-monkey
ENTRYPOINT ["/kube-monkey"]
