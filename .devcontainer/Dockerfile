FROM alpine:3.23.3 AS fetcher

RUN \
  apk add --no-cache curl unzip && \
  mkdir -p /tmp/fetcher && cd /tmp/fetcher && \
  # Fetch dprint
  curl -fsSL -o dprint.zip https://github.com/dprint/dprint/releases/download/0.51.1/dprint-x86_64-unknown-linux-gnu.zip && \
  unzip ./dprint.zip && chmod +x ./dprint && \
  # Fetch task
  curl -fsSL -o task.tar.gz https://github.com/go-task/task/releases/download/v3.48.0/task_linux_amd64.tar.gz && \
  tar -xzf task.tar.gz && chmod +x ./task

# Use golang as the base image
FROM golang:1.26-trixie

# Install golangci-lint
COPY --from=golangci/golangci-lint:v2.8.0 /usr/bin/golangci-lint /usr/local/bin/golangci-lint

# Install dprint and task
COPY --from=fetcher /tmp/fetcher/dprint /usr/local/bin/dprint
COPY --from=fetcher /tmp/fetcher/task /usr/local/bin/task

# Set environment variables
ENV DEBIAN_FRONTEND="noninteractive"

RUN set -e && \
    # Update and install system dependencies
    apt-get update -qq && \
    apt-get install -yqq --no-install-recommends \
        git wget curl zip unzip p7zip-full tree ripgrep && \
    # Clean up
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Add the startup script on every bash session
RUN rm -f /root/.bashrc
COPY .devcontainer/.bashrc /root/.bashrc

# Command just to keep the container running
CMD ["sleep", "infinity"]
