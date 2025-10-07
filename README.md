# SSH Server Implementation

A simple SSH server implementation in Go that supports both password and public key authentication, with full PTY (pseudo-terminal) support for interactive shell sessions.

## Features

- **Dual Authentication**: Supports both password-based and public key authentication
- **PTY Support**: Full pseudo-terminal support for interactive shell sessions
- **Command Execution**: Supports both interactive shells and direct command execution
- **Window Resizing**: Dynamic terminal window resizing support
- **Exit Status**: Proper SSH exit status reporting
- **Session Management**: Handles multiple concurrent SSH connections

## Prerequisites

- Go 1.25.1 or later
- SSH key pair (for key-based authentication)

## Setup

### 1. Generate SSH Key Pair

If you don't have SSH keys, generate them:

```bash
ssh-keygen -t rsa -b 2048 -f id_rsa -N ""
```

This will create:
- `id_rsa` (private key for the server)
- `id_rsa.pub` (public key for client authentication)

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Run the Server

```bash
go run main.go
```

The server will start listening on `0.0.0.0:2222`.

## Configuration

The server configuration is defined in `main.go`:

```go
const (
    serverAddr      = "0.0.0.0:2222"  // Server address and port
    allowedUser     = "testuser"       // Username for password auth
    allowedPassword = "secret123"      // Password for authentication
)
```

## Authentication Methods

### Password Authentication
- **Username**: `testuser`
- **Password**: `secret123`

### Public Key Authentication
- Uses the public key from `id_rsa.pub`
- The corresponding private key must be used by the SSH client

## Usage Examples

### Connect with Password Authentication

```bash
ssh -p 2222 testuser@localhost
# Enter password: secret123
```

### Connect with Public Key Authentication

```bash
ssh -i id_rsa -p 2222 testuser@localhost
```

### Connect from Remote Machine

Replace `localhost` with your machine's IP address:

```bash
ssh -p 2222 testuser@192.168.1.36
```

## Supported SSH Features

- **Session Channels**: Interactive shell sessions
- **PTY Requests**: Pseudo-terminal allocation
- **Window Changes**: Dynamic terminal resizing
- **Shell Requests**: Interactive shell spawning
- **Exec Requests**: Direct command execution
- **Exit Status**: Proper exit code reporting

## Security Notes

⚠️ **This is a demonstration server and should NOT be used in production without proper security hardening:**

- Default credentials are hardcoded
- No rate limiting or connection throttling
- No logging of authentication attempts
- No firewall or access control beyond basic authentication

## Dependencies

- `github.com/creack/pty` - PTY (pseudo-terminal) support
- `golang.org/x/crypto` - SSH protocol implementation

## Project Structure

```
├── main.go          # Main SSH server implementation
├── go.mod           # Go module definition
├── go.sum           # Dependency checksums
├── id_rsa           # Server private key (generate if missing)
├── id_rsa.pub       # Client public key for authentication
└── README.md        # This file
```

## Troubleshooting

### Server won't start
- Ensure port 2222 is not in use by another process
- Check that `id_rsa` private key file exists
- Verify Go version compatibility

### Authentication fails
- For password auth: verify username is `testuser` and password is `secret123`
- For key auth: ensure `id_rsa.pub` exists and matches your client's private key
- Check file permissions on key files

### Connection refused
- Verify the server is running and listening on the correct port
- Check firewall settings if connecting from a remote machine
- Ensure you're using the correct IP address and port (2222)

## License

This project is for educational purposes. Use at your own risk.
