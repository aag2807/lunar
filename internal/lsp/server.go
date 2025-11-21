package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

// Server represents the LSP server
type Server struct {
	initialized bool
	shutdown    bool
	documents   *DocumentManager
	diagnostics *DiagnosticsEngine
	rootURI     string
	logger      *log.Logger
	reader      *bufio.Reader
	writer      io.Writer
}

// NewServer creates a new LSP server
func NewServer() *Server {
	return &Server{
		documents:   NewDocumentManager(),
		diagnostics: NewDiagnosticsEngine(),
		logger:      log.New(os.Stderr, "[lunar-lsp] ", log.LstdFlags),
		reader:      bufio.NewReader(os.Stdin),
		writer:      os.Stdout,
	}
}

// Run starts the LSP server main loop
func (s *Server) Run() error {
	s.logger.Println("Lunar LSP server starting...")

	for {
		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				s.logger.Println("Connection closed")
				return nil
			}
			s.logger.Printf("Error reading message: %v", err)
			continue
		}

		if err := s.handleMessage(msg); err != nil {
			s.logger.Printf("Error handling message: %v", err)
		}

		if s.shutdown {
			s.logger.Println("Server shutting down...")
			return nil
		}
	}
}

// readMessage reads a JSON-RPC message from stdin
func (s *Server) readMessage() (json.RawMessage, error) {
	// Read headers
	var contentLength int
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break // End of headers
		}

		if strings.HasPrefix(line, "Content-Length:") {
			lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, err = strconv.Atoi(lengthStr)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %v", err)
			}
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	// Read content
	content := make([]byte, contentLength)
	_, err := io.ReadFull(s.reader, content)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(content), nil
}

// sendResponse sends a JSON-RPC response
func (s *Server) sendResponse(id interface{}, result interface{}) error {
	response := ResponseMessage{
		Message: Message{JSONRPC: "2.0"},
		ID:      id,
		Result:  result,
	}
	return s.sendMessage(response)
}

// sendError sends a JSON-RPC error response
func (s *Server) sendError(id interface{}, code int, message string) error {
	response := ResponseMessage{
		Message: Message{JSONRPC: "2.0"},
		ID:      id,
		Error: &ResponseError{
			Code:    code,
			Message: message,
		},
	}
	return s.sendMessage(response)
}

// sendNotification sends a JSON-RPC notification
func (s *Server) sendNotification(method string, params interface{}) error {
	notification := NotificationMessage{
		Message: Message{JSONRPC: "2.0"},
		Method:  method,
		Params:  params,
	}
	return s.sendMessage(notification)
}

// sendMessage sends a JSON-RPC message to stdout
func (s *Server) sendMessage(msg interface{}) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
	if _, err := s.writer.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := s.writer.Write(content); err != nil {
		return err
	}

	return nil
}

// handleMessage handles an incoming JSON-RPC message
func (s *Server) handleMessage(content json.RawMessage) error {
	// Parse the base message to get method
	var baseMsg struct {
		ID     interface{} `json:"id"`
		Method string      `json:"method"`
	}

	if err := json.Unmarshal(content, &baseMsg); err != nil {
		return err
	}

	// Check if server is initialized for non-initialize requests
	if !s.initialized && baseMsg.Method != "initialize" && baseMsg.Method != "exit" {
		if baseMsg.ID != nil {
			return s.sendError(baseMsg.ID, ServerNotInitialized, "Server not initialized")
		}
		return nil
	}

	// Route to appropriate handler
	switch baseMsg.Method {
	// Lifecycle
	case "initialize":
		return s.handleInitialize(content)
	case "initialized":
		return s.handleInitialized()
	case "shutdown":
		return s.handleShutdown(baseMsg.ID)
	case "exit":
		return s.handleExit()

	// Document synchronization
	case "textDocument/didOpen":
		return s.handleDidOpen(content)
	case "textDocument/didChange":
		return s.handleDidChange(content)
	case "textDocument/didClose":
		return s.handleDidClose(content)
	case "textDocument/didSave":
		return s.handleDidSave(content)

	// Language features
	case "textDocument/hover":
		return s.handleHover(content, baseMsg.ID)
	case "textDocument/definition":
		return s.handleDefinition(content, baseMsg.ID)
	case "textDocument/completion":
		return s.handleCompletion(content, baseMsg.ID)
	case "textDocument/references":
		return s.handleReferences(content, baseMsg.ID)
	case "textDocument/rename":
		return s.handleRename(content, baseMsg.ID)
	case "textDocument/codeAction":
		return s.handleCodeAction(content, baseMsg.ID)
	case "textDocument/inlayHint":
		return s.handleInlayHint(content, baseMsg.ID)

	default:
		s.logger.Printf("Unhandled method: %s", baseMsg.Method)
		if baseMsg.ID != nil {
			return s.sendError(baseMsg.ID, MethodNotFound, fmt.Sprintf("Method not found: %s", baseMsg.Method))
		}
	}

	return nil
}

// publishDiagnostics publishes diagnostics for a document
func (s *Server) publishDiagnostics(uri string) {
	content := s.documents.GetContent(uri)
	if content == "" {
		return
	}

	diagnostics := s.diagnostics.Analyze(uri, content)

	params := PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}

	if err := s.sendNotification("textDocument/publishDiagnostics", params); err != nil {
		s.logger.Printf("Error publishing diagnostics: %v", err)
	}
}
