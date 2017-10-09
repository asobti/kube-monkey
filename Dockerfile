FROM ubuntu
RUN if (dpkg -l | grep -cq tzdata); then \
        echo "tzdata package already installed!"; \
    else \
        apt-get update && apt-get install tzdata -y --no-install-recommends apt-utils; \
    fi
COPY kube-monkey /kube-monkey
