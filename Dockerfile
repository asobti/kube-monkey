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
