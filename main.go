package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"

	pty "github.com/creack/pty"
	"golang.org/x/crypto/ssh"
)

const (
	serverAddr      = "0.0.0.0:2222"
	allowedUser     = "testuser"
	allowedPassword = "secret123"
)

func main() {
	// Load server's private key (generate one if needed)
	privateBytes, err := os.ReadFile("id_rsa")
	if err != nil {
		log.Fatalf("Failed to load private key (id_rsa): %v", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// Load allowed public key for key-based auth
	authorizedKeyBytes, err := os.ReadFile("id_rsa.pub")
	if err != nil {
		log.Printf("Public key not found, key-based auth will be disabled: %v", err)
	}

	// SSH server config
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == allowedUser && string(pass) == allowedPassword {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeyBytes == nil {
				return nil, fmt.Errorf("no public key auth configured")
			}
			authorizedKey, _, _, _, err := ssh.ParseAuthorizedKey(authorizedKeyBytes)
			if err != nil {
				return nil, fmt.Errorf("invalid public key format")
			}
			if string(key.Marshal()) == string(authorizedKey.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}
	config.AddHostKey(private)

	// Start listening
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", serverAddr, err)
	}
	log.Printf("SSH server listening on %s", serverAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection: %v", err)
			continue
		}

		go handleConn(conn, config)
	}
}

func handleConn(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Handshake failed: %v", err)
		return
	}
	log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
			continue
		}

		go func(ch ssh.Channel, reqs <-chan *ssh.Request) {
			defer ch.Close()

			var (
				ptyRequested bool
				ptyCols      uint32
				ptyRows      uint32
				ptyFile      *os.File
			)

			for req := range reqs {
				switch req.Type {
				case "pty-req":
					// Parse PTY request payload: term, cols, rows, width, height, modes
					var p struct {
						Term   string
						Cols   uint32
						Rows   uint32
						Width  uint32
						Height uint32
						Modes  []byte
					}
					if err := ssh.Unmarshal(req.Payload, &p); err != nil {
						req.Reply(false, nil)
						continue
					}
					ptyRequested = true
					ptyCols = p.Cols
					ptyRows = p.Rows
					req.Reply(true, nil)

				case "window-change":
					// cols, rows, width, height
					var wc struct {
						Cols   uint32
						Rows   uint32
						Width  uint32
						Height uint32
					}
					if err := ssh.Unmarshal(req.Payload, &wc); err == nil {
						ptyCols = wc.Cols
						ptyRows = wc.Rows
						if ptyFile != nil {
							_ = pty.Setsize(ptyFile, &pty.Winsize{Cols: uint16(ptyCols), Rows: uint16(ptyRows)})
						}
					}
					// do not send a reply to window-change per RFC

				case "shell":
					if len(req.Payload) != 0 {
						// We only support default shell (no command payload)
						req.Reply(false, nil)
						continue
					}

					// Start a real shell
					// Prefer bash if available, fall back to sh
					shellPath := "/bin/bash"
					if _, err := os.Stat(shellPath); err != nil {
						shellPath = "/bin/sh"
					}

					cmd := exec.Command(shellPath, "-l")

					if ptyRequested {
						f, err := pty.Start(cmd)
						if err != nil {
							req.Reply(false, nil)
							continue
						}
						ptyFile = f
						// Set initial window size if provided
						if ptyCols > 0 && ptyRows > 0 {
							_ = pty.Setsize(f, &pty.Winsize{Cols: uint16(ptyCols), Rows: uint16(ptyRows)})
						}

						req.Reply(true, nil)

						// Pipe data between SSH channel and PTY
						go func() { _, _ = io.Copy(f, ch) }()
						go func() { _, _ = io.Copy(ch, f) }()

						// Wait for the shell to exit
						if err := cmd.Wait(); err != nil {
							// send exit status if possible
							if exitErr, ok := err.(*exec.ExitError); ok {
								if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
									sendExitStatus(ch, status.ExitStatus())
								}
							}
						} else {
							sendExitStatus(ch, 0)
						}
						return
					}

					// Non-PTY fallback: run interactive sh and connect pipes
					stdin, _ := cmd.StdinPipe()
					stdout, _ := cmd.StdoutPipe()
					stderr, _ := cmd.StderrPipe()
					if err := cmd.Start(); err != nil {
						req.Reply(false, nil)
						continue
					}
					req.Reply(true, nil)
					go func() { _, _ = io.Copy(stdin, ch) }()
					go func() { _, _ = io.Copy(ch, stdout) }()
					go func() { _, _ = io.Copy(ch.Stderr(), stderr) }()
					if err := cmd.Wait(); err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
								sendExitStatus(ch, status.ExitStatus())
							}
						}
					} else {
						sendExitStatus(ch, 0)
					}
					return

				case "exec":
					// Execute a specific command without PTY
					var ex struct{ Command string }
					if err := ssh.Unmarshal(req.Payload, &ex); err != nil {
						req.Reply(false, nil)
						continue
					}
					cmd := exec.Command("/bin/sh", "-c", ex.Command)
					cmd.Stdin = ch
					cmd.Stdout = ch
					cmd.Stderr = ch.Stderr()
					if err := cmd.Start(); err != nil {
						req.Reply(false, nil)
						continue
					}
					req.Reply(true, nil)
					if err := cmd.Wait(); err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
								sendExitStatus(ch, status.ExitStatus())
							}
						}
					} else {
						sendExitStatus(ch, 0)
					}
					return

				default:
					req.Reply(false, nil)
				}
			}
		}(channel, requests)
	}
}

// sendExitStatus sends the SSH-specific exit-status request on the channel.
func sendExitStatus(ch ssh.Channel, status int) {
	// Per RFC 4254, exit-status uses a uint32 payload
	type exitStatus struct{ Status uint32 }
	payload := ssh.Marshal(exitStatus{Status: uint32(status)})
	// Ignore reply; it's a one-way notification
	_, _ = ch.SendRequest("exit-status", false, payload)
}
