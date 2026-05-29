FROM golang:1.22-bookworm
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      libgtk-3-dev \
      libwebkit2gtk-4.0-dev \
      pkg-config \
      gcc \
      g++ && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
