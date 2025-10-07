# SSH Server Implementation Explanation

## Overview
This document explains how SSH (Secure Shell) works and how our Go SSH server implementation handles the protocol.

## What is SSH?
SSH (Secure Shell) is a cryptographic network protocol for secure remote access and data communication. It provides:
- **Encrypted communication** between client and server
- **Authentication** mechanisms (password or public key)
- **Secure remote shell** access
- **File transfer** capabilities
- **Port forwarding** and tunneling

## SSH Protocol Flow

### 1. Connection Establishment
```
Client                    Server
  |                         |
  |---- TCP Connect ------->|
  |                         |
  |<---- SSH Version -------|
  |---- SSH Version ------->|
```

### 2. Key Exchange & Encryption Setup
```
Client                    Server
  |                         |
  |---- Key Exchange ------>|
  |<---- Key Exchange ------|
  |                         |
  |---- Encrypted Data ---->|
  |<---- Encrypted Data ----|
```

### 3. Authentication
```
Client                    Server
  |                         |
  |---- Auth Request ------>|
  |<---- Auth Challenge ----|
  |---- Auth Response ----->|
  |<---- Auth Success ------|
```

### 4. Session Management
```
Client                    Server
  |                         |
  |---- Channel Request ---->|
  |<---- Channel Open ------|
  |                         |
  |---- Shell Request ----->|
  |<---- Shell Granted -----|
  |                         |
  |---- Data -------------->|
  |<---- Data --------------|
```

## Our Go SSH Server Implementation

### Key Components

#### 1. Server Configuration
```go
config := &ssh.ServerConfig{
    PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
        // Verify username/password
    },
    PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
        // Verify public key authentication
    },
}
```

#### 2. Connection Handling
- **TCP Listener**: Binds to `0.0.0.0:2222`
- **SSH Handshake**: Establishes encrypted connection
- **Channel Management**: Handles multiple concurrent sessions

#### 3. Authentication Methods

##### Password Authentication
- Username: `testuser`
- Password: `secret123`
- Server verifies credentials against hardcoded values

##### Public Key Authentication
- Server reads allowed public key from `id_rsa.pub`
- Client proves ownership of matching private key
- More secure than password authentication

#### 4. Session Types

##### Interactive Shell (with PTY)
- Client requests PTY allocation
- Server creates pseudo-terminal
- Handles window resizing
- Provides full interactive experience

##### Command Execution
- Client sends specific command
- Server executes command without PTY
- Returns command output and exit status

## PlantUML Diagrams

### SSH Connection Flow

![file](client-server-communication.png)

<details>

<summary> plantuml</summary>


```plantuml
@startuml
participant "SSH Client" as Client
participant "SSH Server" as Server

Client -> Server: TCP Connect (port 2222)
Server -> Client: SSH Version String
Client -> Server: SSH Version String

Client -> Server: Key Exchange Request
Server -> Client: Key Exchange Response
note right: Establish encrypted tunnel

Client -> Server: Authentication Request
alt Password Auth
    Client -> Server: Username + Password
    Server -> Client: Auth Success/Failure
else Public Key Auth
    Client -> Server: Public Key + Signature
    Server -> Client: Auth Success/Failure
end

Client -> Server: Channel Open Request
Server -> Client: Channel Open Confirmation

Client -> Server: PTY Request (optional)
Server -> Client: PTY Granted

Client -> Server: Shell Request
Server -> Client: Shell Granted

loop Interactive Session
    Client -> Server: User Input
    Server -> Client: Command Output
end

Client -> Server: Exit
Server -> Client: Exit Status
@enduml
```
</details>

### Server Architecture


![file](ssh-server.png)

<details>

<summary> plantuml </summary>

```plantuml
@startuml
package "SSH Server" {
    component "TCP Listener" as Listener
    component "SSH Handshake" as Handshake
    component "Authentication" as Auth
    component "Channel Manager" as ChannelMgr
    component "PTY Handler" as PTY
    component "Shell Executor" as Shell
}

package "External" {
    component "SSH Client" as Client
    component "File System" as FS
    component "Process Manager" as Process
}

Client -> Listener: TCP Connection
Listener -> Handshake: New Connection
Handshake -> Auth: Authentication
Auth -> ChannelMgr: Channel Requests
ChannelMgr -> PTY: PTY Allocation
ChannelMgr -> Shell: Command Execution
Shell -> Process: Process Creation
Shell -> FS: File Operations
@enduml
```
</details>

### Authentication Flow

![file](authentication-flow.png)

<details>

<summary> plantuml </summary>

```plantuml
@startuml
participant "Client" as C
participant "Server" as S

C -> S: Authentication Request
S -> S: Check Auth Method

alt Password Authentication
    C -> S: Username + Password
    S -> S: Verify against allowedUser/allowedPassword
    alt Valid Credentials
        S -> C: Authentication Success
    else Invalid Credentials
        S -> C: Authentication Failed
    end
else Public Key Authentication
    C -> S: Public Key + Signature
    S -> S: Load authorizedKeyBytes from id_rsa.pub
    S -> S: Verify signature with public key
    alt Valid Key
        S -> C: Authentication Success
    else Invalid Key
        S -> C: Authentication Failed
    end
end
@enduml
```
</details>

### PTY and Shell Management

![file](pty-shell-management.png)

<details>

<summary> plantuml </summary>

```plantuml
@startuml
participant "Client" as C
participant "Server" as S
participant "PTY" as P
participant "Shell Process" as SP

C -> S: PTY Request
S -> P: Create PTY
P -> S: PTY File Descriptor
S -> C: PTY Granted

C -> S: Window Change
S -> P: Resize PTY
P -> SP: Update Terminal Size

C -> S: Shell Request
S -> SP: Start Shell Process
SP -> P: Attach to PTY
S -> C: Shell Granted

loop Interactive Session
    C -> S: User Input
    S -> P: Write to PTY
    P -> SP: Send to Shell
    SP -> P: Shell Output
    P -> S: Read from PTY
    S -> C: Send Output
end

SP -> S: Process Exit
S -> C: Exit Status
@enduml
```

</details>

## Key Features of Our Implementation

### 1. **Dual Authentication**
- Password authentication for simplicity
- Public key authentication for security

### 2. **PTY Support**
- Allocates pseudo-terminals for interactive sessions
- Handles window resizing
- Provides proper terminal environment

### 3. **Command Execution**
- Supports both interactive shells and command execution
- Proper exit status reporting
- Error handling and cleanup

### 4. **Concurrent Connections**
- Handles multiple simultaneous connections
- Goroutine-based connection handling
- Proper resource management

## Security Considerations

### 1. **Host Key Verification**
- Server uses private host key for identity
- Clients should verify host key fingerprint

### 2. **Authentication**
- Public key authentication is more secure than passwords
- Consider implementing additional security measures

### 3. **Network Security**
- All communication is encrypted
- Consider firewall rules and access controls

## Usage Examples

### Starting the Server
```bash
# Generate host key if needed
ssh-keygen -t rsa -b 2048 -f id_rsa -N ""

# Run the server
go run main.go
```

### Connecting with Password
```bash
ssh -p 2222 testuser@127.0.0.1
# Enter password: secret123
```

### Connecting with Public Key
```bash
# Copy your public key to id_rsa.pub
ssh -p 2222 testuser@127.0.0.1
```

### Executing Commands
```bash
ssh -p 2222 testuser@127.0.0.1 'ls -la'
```

## Cross-Platform Compatibility

This SSH server implementation works across different operating systems:
- **Windows**: Run with Go installed
- **macOS**: Native support
- **Linux**: Native support

Clients can connect from any SSH-compatible application regardless of the operating system.

## Conclusion

This Go SSH server implementation provides a solid foundation for understanding SSH protocol internals. It demonstrates key concepts like authentication, PTY management, and secure communication while maintaining simplicity for educational purposes.

For production use, consider additional features like:
- User management system
- Logging and auditing
- Advanced security measures
- File transfer capabilities (SFTP)
- Port forwarding support
