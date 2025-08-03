# RCode Development Environment Setup

## Prerequisites

### Required Software
- **Go 1.21+**: The project is written in Go
  - macOS: `brew install go`
  - Ubuntu/Debian: `sudo apt-get install golang-go`
  - Or download from https://golang.org/dl/

- **Make**: For build automation
  - macOS: Comes pre-installed with Xcode Command Line Tools
  - Ubuntu/Debian: `sudo apt-get install build-essential`

- **Git**: Version control
  - macOS: `brew install git`
  - Ubuntu/Debian: `sudo apt-get install git`

### Optional but Recommended
- **golangci-lint**: For linting
  - Install via: `make install-tools`
- **Docker**: For containerized builds (optional)
  - Follow instructions at https://docs.docker.com/get-docker/

## Initial Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/foxytanuki/rcode.git
   cd rcode
   ```

2. **Install Go dependencies**
   ```bash
   go mod download
   # or
   make deps
   ```

3. **Install development tools**
   ```bash
   make install-tools
   ```

4. **Verify setup**
   ```bash
   # Run tests
   make test
   
   # Try building
   make build
   ```

## Project Structure

```
rcode/
├── cmd/                    # Application entry points
│   ├── rcode/             # CLI client
│   └── server/            # HTTP server
├── internal/              # Private application code
│   ├── config/           # Configuration management
│   ├── editor/           # Editor management
│   ├── errors/           # Error handling
│   ├── logger/           # Logging utilities
│   ├── network/          # Network utilities
│   └── security/         # Security utilities
├── pkg/                   # Public libraries
│   └── api/              # API types and contracts
├── scripts/              # Utility scripts
├── docs/                 # Documentation
│   └── _local/          # Local development docs
├── bin/                  # Built binaries (git-ignored)
├── go.mod               # Go module definition
├── go.sum               # Go module checksums
├── Makefile             # Build automation
├── CLAUDE.md            # AI assistant instructions
├── DEVELOPMENT.md       # This file
└── README.md            # Project overview

```

## Development Workflow

### Building

```bash
# Build for current platform
make build

# Build server only
make build-server

# Build client only
make build-client

# Cross-compile for all platforms
make build-all

# Build for specific platform
make build-darwin-amd64  # macOS Intel
make build-darwin-arm64  # macOS Apple Silicon
make build-linux-amd64   # Linux x86_64
make build-linux-arm64   # Linux ARM64
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run benchmarks
make benchmark

# Run short tests only
make test-short
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet

# Tidy dependencies
make tidy
```

### Running Locally

```bash
# Run server
make run-server
# or
go run ./cmd/server

# Run client
make run-client
# or
go run ./cmd/rcode
```

### Installing

```bash
# Install to /usr/local/bin (requires sudo)
make install

# Uninstall
make uninstall
```

## Configuration

### Server Configuration
Default location: `~/.config/rcode/server-config.yaml`

Example:
```yaml
server:
  host: "0.0.0.0"
  port: 3000
  
editors:
  - name: cursor
    command: "cursor --remote ssh-remote+{user}@{host} {path}"
    default: true
  - name: vscode
    command: "code --remote ssh-remote+{user}@{host} {path}"
  - name: nvim
    command: "nvim scp://{user}@{host}/{path}"
    
logging:
  level: info
  file: ~/.local/share/rcode/logs/server.log
```

### Client Configuration
Default location: `~/.config/rcode/config.yaml`

Example:
```yaml
network:
  primary_host: "192.168.1.100"
  fallback_host: "100.64.0.1"  # Tailscale
  timeout: "2s"
  
default_editor: cursor

logging:
  level: info
  file: ~/.local/share/rcode/logs/client.log
```

## Environment Variables

- `RCODE_CONFIG`: Override default config file location
- `RCODE_LOG_LEVEL`: Set log level (debug, info, warn, error)
- `RCODE_HOST`: Override server host (client)
- `RCODE_PORT`: Override server port
- `RCODE_EDITOR`: Override default editor (client)

## Debugging

### Enable Debug Logging
```bash
# Server
RCODE_LOG_LEVEL=debug ./bin/rcode-server

# Client
RCODE_LOG_LEVEL=debug ./bin/rcode
```

### View Logs
```bash
# Server logs
tail -f ~/.local/share/rcode/logs/server.log

# Client logs
tail -f ~/.local/share/rcode/logs/client.log
```

## Common Issues

### Go Module Issues
```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download

# Verify dependencies
go mod verify
```

### Build Issues
```bash
# Clean build artifacts
make clean

# Rebuild
make build
```

### Permission Issues
- Ensure config directories exist and are writable
- Server needs permission to bind to port (default 3000)
- Client needs permission to read SSH environment variables

## Contributing

1. Create a feature branch
   ```bash
   git checkout -b feature/your-feature
   ```

2. Make changes and test
   ```bash
   make test
   make lint
   ```

3. Commit with descriptive message
   ```bash
   git commit -m "feat: add new feature"
   ```

4. Push and create PR
   ```bash
   git push origin feature/your-feature
   ```

## Useful Make Commands

Run `make help` to see all available commands:

```bash
make help        # Show help
make build       # Build binaries
make test        # Run tests
make lint        # Run linter
make fmt         # Format code
make clean       # Clean artifacts
make install     # Install binaries
make run-server  # Run server
make run-client  # Run client
```

## Next Steps

After setting up the development environment:

1. Review `docs/_local/TASK.md` for implementation tasks
2. Check `CLAUDE.md` for project architecture details
3. Start with Phase 1 tasks in order
4. Write tests for each component
5. Update documentation as you go