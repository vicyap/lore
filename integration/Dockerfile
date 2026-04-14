FROM golang:1.24-bookworm

RUN apt-get update && apt-get install -y git curl && rm -rf /var/lib/apt/lists/*

# Install Node.js for claude CLI
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get install -y nodejs && \
    npm install -g @anthropic-ai/claude-code

WORKDIR /app
COPY . .

RUN go build -o /usr/local/bin/lore ./cmd/lore

COPY integration/docker_e2e.sh /docker_e2e.sh
RUN chmod +x /docker_e2e.sh

ENTRYPOINT ["/docker_e2e.sh"]
