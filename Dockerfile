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

COPY kube-monkey /kube-monkey
