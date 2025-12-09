# E2E test environment for cccc
FROM golang:1.25.5-alpine

# Install bash for test scripts
RUN apk add --no-cache bash

# Create test user
RUN adduser -D testuser
USER testuser
WORKDIR /home/testuser

# Pre-populate a realistic ~/.claude structure
COPY --chown=testuser:testuser test/fixtures/claude-home /home/testuser/.claude

# Create project directories that "exist" (active projects)
RUN mkdir -p /home/testuser/Code/active-project
RUN mkdir -p /home/testuser/Code/another-active

# These directories are intentionally NOT created (simulating deleted projects):
# - /home/testuser/Code/deleted-project
# - /home/testuser/old-stuff/archived

# Copy the source code
COPY --chown=testuser:testuser . /app
WORKDIR /app

# Build the binary
RUN go build -o cccc ./cmd/ccc

# Add healthcheck (required by security scanners)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["./cccc", "--help"]

# Run E2E tests
ENTRYPOINT ["go", "test", "-v", "-tags=e2e", "./test/e2e/..."]
