# RCode API Documentation

## Overview

The RCode server exposes a simple HTTP REST API for opening editors on the host machine from remote clients.

Base URL: `http://<host>:3339`

## Authentication

Currently, the API does not require authentication. Security is provided through:
- IP whitelist configuration (optional)
- Running on internal network only
- Rate limiting per IP address

## Endpoints

### 1. Open Editor

Opens a file or directory in an editor on the host machine.

**Endpoint:** `POST /open-editor`

**Request Body:**
```json
{
  "path": "/home/user/project",
  "editor": "cursor",
  "user": "alice",
  "host": "remote-server.example.com",
  "timestamp": 1704067200
}
```

**Fields:**
- `path` (string, required): The file or directory path to open
- `editor` (string, optional): The editor to use. If not specified, uses the default editor
- `user` (string, required): The SSH username on the remote machine
- `host` (string, required): The hostname of the remote machine
- `timestamp` (integer, optional): Unix timestamp of the request

**Success Response (200 OK):**
```json
{
  "success": true,
  "message": "Opened /home/user/project in cursor",
  "editor": "cursor",
  "command": "cursor --remote ssh-remote+alice@remote-server.example.com /home/user/project",
  "timestamp": 1704067201
}
```

**Error Response (400 Bad Request):**
```json
{
  "error": "invalid path specified",
  "code": "INVALID_PATH",
  "details": "Path cannot be empty",
  "timestamp": 1704067201
}
```

**Error Response (404 Not Found):**
```json
{
  "error": "editor not found",
  "code": "EDITOR_NOT_FOUND",
  "details": "Editor 'sublime' is not configured",
  "timestamp": 1704067201
}
```

**Error Codes:**
- `INVALID_REQUEST` - Request format is invalid
- `INVALID_PATH` - Path is invalid or empty
- `MISSING_USER` - User field is missing
- `MISSING_HOST` - Host field is missing
- `INVALID_EDITOR` - Editor name is invalid
- `EDITOR_NOT_FOUND` - Requested editor is not configured
- `EDITOR_UNAVAILABLE` - Editor is not available on the system
- `EDITOR_EXECUTION_ERROR` - Failed to execute editor command

### 2. Health Check

Check if the server is running and healthy.

**Endpoint:** `GET /health`

**Success Response (200 OK):**
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime": 3600,
  "timestamp": 1704067200,
  "started_at": "2024-01-01T00:00:00Z"
}
```

**Fields:**
- `status` (string): "healthy" or "unhealthy"
- `version` (string): Server version
- `uptime` (integer): Server uptime in seconds
- `timestamp` (integer): Unix timestamp
- `started_at` (string): Server start time in RFC3339 format

### 3. List Editors

Get the list of available editors on the host machine.

**Endpoint:** `GET /editors`

**Success Response (200 OK):**
```json
{
  "editors": [
    {
      "name": "cursor",
      "command": "cursor --remote ssh-remote+{user}@{host} {path}",
      "available": true,
      "default": true
    },
    {
      "name": "vscode",
      "command": "code --remote ssh-remote+{user}@{host} {path}",
      "available": true,
      "default": false
    },
    {
      "name": "nvim",
      "command": "nvim scp://{user}@{host}/{path}",
      "available": false,
      "default": false
    }
  ],
  "default_editor": "cursor",
  "timestamp": 1704067200
}
```

**Fields:**
- `editors` (array): List of configured editors
  - `name` (string): Editor identifier
  - `command` (string): Command template with placeholders
  - `available` (boolean): Whether the editor is installed and available
  - `default` (boolean): Whether this is the default editor
- `default_editor` (string): Name of the default editor
- `timestamp` (integer): Unix timestamp

## Error Handling

All error responses follow a consistent format:

```json
{
  "error": "Human-readable error message",
  "code": "MACHINE_READABLE_CODE",
  "details": "Additional context or debugging information",
  "timestamp": 1704067200
}
```

### HTTP Status Codes

- `200 OK` - Request successful
- `400 Bad Request` - Invalid request data
- `404 Not Found` - Requested resource not found
- `405 Method Not Allowed` - HTTP method not supported
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

## Rate Limiting

The server implements rate limiting per IP address:
- Maximum 100 requests per minute per IP
- Returns 429 status when limit is exceeded

## Command Templates

Editor commands use template placeholders that are replaced at runtime:

- `{user}` - SSH username from the remote machine
- `{host}` - Hostname of the remote machine
- `{path}` - File or directory path to open

Example: `cursor --remote ssh-remote+{user}@{host} {path}`
Becomes: `cursor --remote ssh-remote+alice@server.com /home/project`

## Usage Examples

### cURL Examples

**Open a file in the default editor:**
```bash
curl -X POST http://192.168.1.100:3339/open-editor \
  -H "Content-Type: application/json" \
  -d '{
    "path": "/home/alice/project",
    "user": "alice",
    "host": "dev-server"
  }'
```

**Open a file in a specific editor:**
```bash
curl -X POST http://192.168.1.100:3339/open-editor \
  -H "Content-Type: application/json" \
  -d '{
    "path": "/home/alice/project",
    "editor": "vscode",
    "user": "alice",
    "host": "dev-server"
  }'
```

**Check server health:**
```bash
curl http://192.168.1.100:3339/health
```

**List available editors:**
```bash
curl http://192.168.1.100:3339/editors
```

### Python Example

```python
import requests
import json

def open_editor(path, editor=None):
    url = "http://192.168.1.100:3339/open-editor"
    data = {
        "path": path,
        "user": "alice",
        "host": "dev-server"
    }
    if editor:
        data["editor"] = editor
    
    response = requests.post(url, json=data)
    if response.status_code == 200:
        print(f"Successfully opened {path}")
    else:
        error = response.json()
        print(f"Error: {error['error']}")

# Usage
open_editor("/home/alice/project")
open_editor("/home/alice/file.py", editor="vscode")
```

## WebSocket Support (Future)

WebSocket support for real-time notifications is planned for a future version.