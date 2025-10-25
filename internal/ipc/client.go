package ipc

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// Client represents an IPC client
type Client struct {
	socketPath string
}

// NewClient creates a new IPC client
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
	}
}

// SendCommand sends a command to the daemon and returns the response
func (c *Client) SendCommand(command string) (string, error) {
	// Connect to Unix socket
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return "", fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer conn.Close()

	// Send command
	if _, err := conn.Write([]byte(command + "\n")); err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return "", fmt.Errorf("no response from daemon")
}
