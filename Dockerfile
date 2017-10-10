FROM ubuntu
RUN if (dpkg -l | grep -cq tzdata); then \
        echo "tzdata package already installed! Skipping tzdata installation"; \
    else \
        echo "Installing tzdata to avoid go panic caused by missing timezone data"; \
        apt-get update && apt-get install tzdata -y --no-install-recommends apt-utils; \
    fi
COPY kube-monkey /kube-monkey
