package tests

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/mcp"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// startEchoWith registers the given route on a new Echo instance and serves it on 127.0.0.1:0.
// It returns the address (host:port) and a shutdown function.
func startEchoWith(path string, handler echo.HandlerFunc) (addr string, shutdown func(), err error) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET(path, handler)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", func() {}, err
	}
	srv := &http.Server{Handler: e}
	done := make(chan struct{})
	go func() {
		_ = srv.Serve(ln)
		close(done)
	}()

	shutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		<-done
	}
	return ln.Addr().String(), shutdown, nil
}

// wsRWC adapts a WebSocket to io.ReadWriteCloser for sdkmcp.IOTransport on the client side.
type wsRWC struct {
	c   *websocket.Conn
	mu  chan struct{} // simple binary semaphore
	buf []byte
	cls bool
}

func newWSRWC(c *websocket.Conn) *wsRWC { return &wsRWC{c: c, mu: make(chan struct{}, 1)} }

func (w *wsRWC) lock()   { w.mu <- struct{}{} }
func (w *wsRWC) unlock() { <-w.mu }

func (w *wsRWC) Read(p []byte) (int, error) {
	// Quick closed check
	w.lock()
	if w.cls {
		w.unlock()
		return 0, io.EOF
	}
	// Serve any buffered data
	if len(w.buf) > 0 {
		n := copy(p, w.buf)
		w.buf = w.buf[n:]
		w.unlock()
		return n, nil
	}
	w.unlock()

	// Blocking read without holding the lock
	mt, data, err := w.c.ReadMessage()
	if err != nil {
		return 0, err
	}
	if mt != websocket.TextMessage {
		return 0, io.ErrUnexpectedEOF
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	w.lock()
	defer w.unlock()
	if w.cls {
		return 0, io.EOF
	}
	w.buf = append(w.buf, data...)
	n := copy(p, w.buf)
	w.buf = w.buf[n:]
	return n, nil
}

func (w *wsRWC) Write(p []byte) (int, error) {
	w.lock()
	defer w.unlock()
	if w.cls {
		return 0, io.ErrClosedPipe
	}
	if err := w.c.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *wsRWC) Close() error {
	w.lock()
	defer w.unlock()
	if w.cls {
		return nil
	}
	w.cls = true
	_ = w.c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	return w.c.Close()
}

func TestMCP_WebSocket_Server_ListAndExecute(t *testing.T) {
	path := "/mcp"
	addr, shutdown, err := startEchoWith(path, mcp.HandleWS)
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer shutdown()

	// Dial with go-sdk client
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test-client", Version: "0"}, nil)
	url := "ws://" + addr + path
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	rwc := newWSRWC(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sess, err := client.Connect(ctx, &sdkmcp.IOTransport{Reader: rwc, Writer: rwc}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	lt, err := sess.ListTools(ctx, &sdkmcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	have := map[string]bool{}
	for _, tl := range lt.Tools {
		have[tl.Name] = true
	}
	if !have["ping"] || !have["execute"] {
		t.Fatalf("expected ping and execute, got: %#v", have)
	}

	res, err := sess.CallTool(ctx, &sdkmcp.CallToolParams{Name: "execute", Arguments: map[string]any{"code": "add(1,2)"}})
	if err != nil {
		t.Fatalf("call execute: %v", err)
	}
	if res.IsError {
		t.Fatalf("execute returned error: %#v", res)
	}
	var got string
	if len(res.Content) > 0 {
		if tc, ok := res.Content[0].(*sdkmcp.TextContent); ok {
			got = tc.Text
		}
	}
	if got != "3" {
		t.Fatalf("expected '3', got %q", got)
	}
	_ = rwc.Close()
}
