# RCode - Remote Code Launcher

A Go-based system that allows launching host machine code editors from SSH-connected remote machines without requiring SSH server on the host.

## Overview

RCode enables seamless code editing when working on remote servers by allowing you to open files from the remote machine directly in your local editor (VSCode, Cursor, Neovim, etc.).

## Architecture

```
Remote Machine → HTTP POST → Host Server → Launch Editor
   (rcode CLI)     (JSON)    (rcode-server)  (cursor/vscode/nvim)
```

## Features

- Launch local editors from remote SSH sessions
- Support for multiple editors (Cursor, VSCode, Neovim)
- Automatic network fallback (LAN → Tailscale)
- No SSH server required on host machine
- Cross-platform support (macOS, Linux)

## Quick Start

### Prerequisites

- Go 1.21+ (for building from source)
- Make (for build automation)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/foxytanuki/rcode.git
cd rcode
```

2. Build the binaries:
```bash
make build
```

3. Install to system:
```bash
make install  # Installs to /usr/local/bin
```

### Usage

#### On Host Machine (Mac/Linux)

1. Start the server:
```bash
rcode-server
```

The server will listen on port 3000 by default.

#### On Remote Machine (via SSH)

1. Open a file or directory in your local editor:
```bash
rcode /path/to/project
```

The client will automatically detect SSH connection information and send a request to your host machine to open the specified path.

## Configuration

### Server Configuration
Location: `~/.config/rcode/server-config.yaml`

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
```

### Client Configuration
Location: `~/.config/rcode/config.yaml`

```yaml
network:
  primary_host: "192.168.1.100"  # Your host's LAN IP
  fallback_host: "100.64.0.1"     # Tailscale IP
  timeout: "2s"

default_editor: cursor
```

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for development environment setup and contribution guidelines.

## Project Status

This project is under active development. See [docs/_local/TASK.md](docs/_local/TASK.md) for the implementation roadmap.

### Current Phase: MVP Development
- [x] Project setup and foundation
- [ ] Shared components (pkg/api)
- [ ] Configuration management
- [ ] Logger component
- [ ] HTTP server implementation
- [ ] Editor management
- [ ] CLI client implementation
- [ ] Basic testing and documentation

## License

MIT

## Author

[@foxytanuki](https://github.com/foxytanuki)