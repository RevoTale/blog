FROM mcr.microsoft.com/devcontainers/go
RUN apt-get update && apt-get install -y curl

# Install Task to a directory already in the PATH (like /usr/local/bin or a user's local bin)
# The example below uses /usr/local/bin for system-wide access
RUN sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin