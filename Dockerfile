########################
### Builder          ###
########################
FROM golang:1.18 as builder
RUN mkdir -p /kube-monkey
COPY ./ /kube-monkey/
WORKDIR /kube-monkey
RUN make build

########################
### Final            ###
########################
FROM scratch
COPY --from=builder /kube-monkey/kube-monkey /kube-monkey
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENTRYPOINT ["/kube-monkey"]
