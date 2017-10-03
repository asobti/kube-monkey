FROM ubuntu
RUN apt-get update
RUN apt-get install tzdata -y
COPY kube-monkey /kube-monkey
