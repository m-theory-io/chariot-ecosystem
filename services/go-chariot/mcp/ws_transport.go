package mcp

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// websocketUpgrader returns a configured websocket.Upgrader.
func websocketUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
		// EnableCompression can be toggled if needed
	}
}

// wsReadWriteCloser adapts a Gorilla websocket.Conn to an io.ReadWriteCloser
// using newline-delimited JSON frames compatible with mcp.IOTransport.
// Each Write sends a single text frame containing the provided bytes.
// Each Read reads a full text frame and serves it to the reader, appending a trailing newline.
// This allows mcp's ioConn decoder to process messages correctly.
// Note: this implementation assumes each frame contains a single JSON message or batch.
// Batching semantics are handled by the higher-level ioConn.

type wsReadWriteCloser struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	rbuf   bytes.Buffer
	closed bool
}

func newWSReadWriteCloser(conn *websocket.Conn) io.ReadWriteCloser {
	return &wsReadWriteCloser{conn: conn}
}

func (w *wsReadWriteCloser) Read(p []byte) (int, error) {
	// Avoid holding the mutex during a potentially blocking network read to prevent deadlocks
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return 0, io.EOF
	}
	if w.rbuf.Len() > 0 {
		n, err := w.rbuf.Read(p)
		w.mu.Unlock()
		return n, err
	}
	w.mu.Unlock()

	// Read next message from websocket without holding lock
	msgType, data, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	if msgType != websocket.TextMessage {
		return 0, errors.New("unsupported websocket message type")
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, io.EOF
	}
	w.rbuf.Write(data)
	return w.rbuf.Read(p)
}

func (w *wsReadWriteCloser) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	// Write entire payload as a single text frame
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *wsReadWriteCloser) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	// Attempt a close control frame, then close underlying
	_ = w.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	return w.conn.Close()
}
