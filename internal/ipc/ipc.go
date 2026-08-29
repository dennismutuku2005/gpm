package ipc

import (
	"encoding/json"
	"net"

	"github.com/dennismutuku2005/gpm/internal/config"
)

// Request defines the structure of messages sent from CLI to Daemon.
type Request struct {
	Command      string               `json:"command"` // start, stop, restart, delete, list, status, logs, enable, disable, shutdown
	Name         string               `json:"name"`
	TargetConfig *config.ProcessConfig `json:"target_config,omitempty"`
	Lines        int                  `json:"lines,omitempty"`  // for tailing logs
	Query        string               `json:"query,omitempty"`  // for filtering logs
}

// Response defines the structure of messages sent from Daemon to CLI.
type Response struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// WriteRequest writes a JSON request to a connection.
func WriteRequest(conn net.Conn, req *Request) error {
	encoder := json.NewEncoder(conn)
	return encoder.Encode(req)
}

// ReadRequest reads a JSON request from a connection.
func ReadRequest(conn net.Conn) (*Request, error) {
	var req Request
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

// WriteResponse writes a JSON response to a connection.
func WriteResponse(conn net.Conn, resp *Response) error {
	encoder := json.NewEncoder(conn)
	return encoder.Encode(resp)
}

// ReadResponse reads a JSON response from a connection.
func ReadResponse(conn net.Conn) (*Response, error) {
	var resp Response
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
