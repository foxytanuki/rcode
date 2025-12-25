# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-12-25

### Added
- System service support for automatic startup on login
  - macOS: launchd user agent support
  - Linux: systemd user service support
  - Service management commands: `-install-service`, `-uninstall-service`, `-start-service`, `-stop-service`, `-status-service`
  - Automatic restart on crash
  - Service logs in `~/.local/share/rcode/logs/service.log`

### Changed
- Improved SSH host detection: `ClientIP` from `SSH_CONNECTION` is now prioritized over config file `ssh_host`
  - This ensures the actual SSH connection source IP is used for editor connections
  - Config file `ssh_host` is now only used as a fallback when `ClientIP` is not available

### Fixed
- SSH host detection now correctly uses the IP address from the current SSH session instead of a fixed hostname from config

## [0.0.1] - 2025-08-03

### Added
- Initial release of Remote Code Launcher (rcode)
- HTTP server (`rcode-server`) for host machines to receive editor launch requests
- CLI client (`rcode`) for remote machines to send editor open requests
- Support for multiple editors: VS Code, Cursor, Neovim
- Network fallback from LAN to Tailscale
- YAML configuration for both client and server
- Automatic SSH connection detection
- Editor availability checking
- Structured logging with file rotation
- Comprehensive test coverage
- Security features: command injection prevention, path validation

### Supported Platforms
- macOS (Intel/Apple Silicon)
- Linux (x86_64/ARM64)

### Supported Editors
- Visual Studio Code
- Cursor
- Neovim (via SSH)
