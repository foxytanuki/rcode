# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] - 2025-12-25

### Fixed
- SSH host detection now correctly prioritizes `ssh_host` from config file over `ClientIP` from `SSH_CONNECTION`
  - Config file `ssh_host` is now checked before auto-detected `ClientIP`
  - This allows users to explicitly specify the correct IP/hostname for editor connections
  - Fixed issue where `192.168.1.34` (SSH client IP) was used instead of configured `192.168.1.40` (remote machine IP)

## [0.1.0] - 2025-12-25

### Added
- System service support for automatic startup on login
  - macOS: launchd user agent support
  - Linux: systemd user service support
  - Service management commands: `-install-service`, `-uninstall-service`, `-start-service`, `-stop-service`, `-status-service`
  - Automatic restart on crash
  - Service logs in `~/.local/share/rcode/logs/service.log`

### Changed
- SSH host detection priority updated: config file `ssh_host` now takes priority over auto-detected `ClientIP`
  - This allows users to explicitly specify the correct IP/hostname for editor connections
  - `ClientIP` from `SSH_CONNECTION` is used as fallback when `ssh_host` is not configured

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
