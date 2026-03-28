package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// StdioTransport handles JSON-RPC over stdin/stdout.
type StdioTransport struct {
	reader *bufio.Scanner
	writer io.Writer
}

// NewStdioTransport creates a transport from reader/writer.
func NewStdioTransport(reader io.Reader, writer io.Writer) *StdioTransport {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	return &StdioTransport{
		reader: scanner,
		writer: writer,
	}
}

// ReadRequest reads the next JSON-RPC request from the transport.
func (t *StdioTransport) ReadRequest() (JSONRPCRequest, error) {
	if !t.reader.Scan() {
		if err := t.reader.Err(); err != nil {
			return JSONRPCRequest{}, fmt.Errorf("reading request: %w", err)
		}
		return JSONRPCRequest{}, io.EOF
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(t.reader.Bytes(), &req); err != nil {
		return JSONRPCRequest{}, fmt.Errorf("parsing request: %w", err)
	}
	return req, nil
}

// WriteResponse writes a JSON-RPC response to the transport.
func (t *StdioTransport) WriteResponse(resp JSONRPCResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}

	_, err = fmt.Fprintf(t.writer, "%s\n", data)
	return err
}
