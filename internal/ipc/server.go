package ipc

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// CommandHandler is a function that handles IPC commands
type CommandHandler func(command string) string

// Server represents an IPC server using Unix sockets
type Server struct {
	socketPath string
	listener   net.Listener
	handler    CommandHandler
}

// NewServer creates a new IPC server
func NewServer(socketPath string, handler CommandHandler) *Server {
	return &Server{
		socketPath: socketPath,
		handler:    handler,
	}
}

// Start starts the IPC server
func (s *Server) Start() error {
	// Remove old socket if it exists
	os.Remove(s.socketPath)

	// Create socket directory
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	s.listener = listener

	// Set socket permissions
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	fmt.Printf("🔌 IPC server listening on: %s\n", s.socketPath)

	// Accept connections in background
	go s.acceptConnections()

	return nil
}

// acceptConnections accepts and handles incoming connections
func (s *Server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// Server closed
			return
		}

		// Handle connection in goroutine
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read command
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())

		// Process command
		response := s.handler(command)

		// Send response
		conn.Write([]byte(response + "\n"))
	}
}

// Stop stops the IPC server
func (s *Server) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
	os.Remove(s.socketPath)
}
