# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Remote Code Launcher (rcode) - A Go-based system that allows launching host machine code editors from SSH-connected remote machines without requiring SSH server on the host.

## Architecture

### System Components

1. **rcode-server** (Host/Mac side)
   - HTTP server running on port 3339
   - Receives editor launch requests from remote clients
   - Executes editor commands locally
   - Endpoints: `/open-editor` (POST), `/health` (GET), `/editors` (GET)

2. **rcode** CLI (Remote/Ubuntu side)
   - Sends HTTP requests to host server
   - Network fallback: Primary LAN (192.168.1.x) → Tailscale
   - Auto-detects current directory and SSH connection info

### Communication Flow
```
Remote Machine → HTTP POST → Host Server → Launch Editor
   (rcode CLI)     (JSON)    (rcode-server)  (cursor/vscode/nvim)
```

## Development Commands

### Project Structure (Planned)
```bash
# Build commands (when implemented)
make build-all                                    # Cross-compile all binaries
GOOS=darwin GOARCH=amd64 go build -o bin/rcode-server-darwin ./cmd/server
GOOS=linux GOARCH=amd64 go build -o bin/rcode-linux ./cmd/rcode

# Run tests
go test ./...                                     # Run all tests
go test -v ./internal/network                     # Test specific package
go test -run TestNetworkFallback                  # Run specific test

# Linting
golangci-lint run                                 # Run linter
go fmt ./...                                      # Format code
go vet ./...                                      # Run go vet
```

## Key Implementation Details

### Package Structure
- `cmd/rcode/` - CLI client implementation
- `cmd/server/` - HTTP server implementation  
- `internal/config/` - YAML config management (uses gopkg.in/yaml.v3)
- `internal/editor/` - Editor command templating and execution
- `internal/network/` - Network fallback logic and HTTP client
- `internal/logger/` - Structured logging
- `pkg/api/` - Shared API types (OpenRequest, OpenResponse)

### Configuration Files
- Client: `~/.config/rcode/config.yaml`
- Server: `~/.config/rcode/server-config.yaml`
- Logs: `~/.local/share/rcode/logs/`

### Critical Data Structures
- `EditorConfig`: Defines editor command templates with `{user}`, `{host}`, `{path}` placeholders
- `NetworkConfig`: Manages primary/fallback hosts with timeout settings
- `OpenRequest/Response`: API contract between client and server

### Network Fallback Implementation
1. Try primary host (LAN) with 2s timeout
2. On failure, try Tailscale with 2s timeout  
3. On all failures, display manual command to user

### Editor Command Templates
Templates use placeholders that get replaced at runtime:
- `{user}` - SSH username
- `{host}` - Remote hostname
- `{path}` - Directory path to open

Example: `cursor --remote ssh-remote+{user}@{host} {path}`

## Testing Strategy

### Unit Tests
- Network fallback logic in `internal/network/`
- Config parsing and validation in `internal/config/`
- Editor command templating in `internal/editor/`

### Integration Tests
- Full client-server communication flow
- Editor launch verification
- Network timeout handling

## Security Considerations

- Command injection prevention: Use `exec.Command` with separate args, never shell expansion
- Path validation: Prevent directory traversal attacks
- IP whitelist: Implement allowed IP ranges in server config
- No authentication by design (internal network only)

## Dependencies

### External packages
- `gopkg.in/yaml.v3` - YAML configuration
- `github.com/spf13/cobra` - CLI framework (planned)

### Standard library usage
- `net/http` - HTTP server/client
- `encoding/json` - API serialization
- `os/exec` - Editor command execution
- `context` - Timeout management