package handlers

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/labstack/echo/v4"
)

// Request payload for saving a diagram
type diagramSaveReq struct {
	Name    string          `json:"name"`
	Content json.RawMessage `json:"content"` // raw Visual DSL diagram JSON
}

// Metadata for listed diagrams
type diagramMeta struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

func sanitizeDiagramName(name string) (string, error) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", errors.New("empty diagram name")
	}
	// Prevent path traversal by removing any path separators
	n = strings.ReplaceAll(n, string(os.PathSeparator), "_")
	// Drop extension if present, enforce .json
	n = strings.TrimSuffix(n, ".json")
	return n + ".json", nil
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// ListDiagrams returns diagram list in configured DiagramPath
func (h *Handlers) ListDiagrams(c echo.Context) error {
	base := cfg.ChariotConfig.DiagramPath
	if strings.TrimSpace(base) == "" {
		return c.JSON(http.StatusServiceUnavailable, ResultJSON{Result: "ERROR", Data: "diagram path not configured"})
	}
	if err := ensureDir(base); err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	out := make([]diagramMeta, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if info, err := e.Info(); err == nil {
			out = append(out, diagramMeta{
				Name:     strings.TrimSuffix(e.Name(), ".json"),
				Size:     info.Size(),
				Modified: info.ModTime(),
			})
		}
	}
	return c.JSON(http.StatusOK, ResultJSON{Result: "OK", Data: out})
}

// GetDiagram returns a single diagram JSON by name
func (h *Handlers) GetDiagram(c echo.Context) error {
	name := c.Param("name")
	base := cfg.ChariotConfig.DiagramPath
	if strings.TrimSpace(base) == "" {
		return c.JSON(http.StatusServiceUnavailable, ResultJSON{Result: "ERROR", Data: "diagram path not configured"})
	}
	file, err := sanitizeDiagramName(name)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	data, err := os.ReadFile(filepath.Join(base, file))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c.JSON(http.StatusNotFound, ResultJSON{Result: "ERROR", Data: "not found"})
		}
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	// return raw content
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	_, _ = c.Response().Write(data)
	return nil
}

// SaveDiagram persists/overwrites a diagram JSON by name
func (h *Handlers) SaveDiagram(c echo.Context) error {
	base := cfg.ChariotConfig.DiagramPath
	if strings.TrimSpace(base) == "" {
		return c.JSON(http.StatusServiceUnavailable, ResultJSON{Result: "ERROR", Data: "diagram path not configured"})
	}
	var req diagramSaveReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "invalid request"})
	}
	file, err := sanitizeDiagramName(req.Name)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	if err := ensureDir(base); err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	if len(req.Content) == 0 {
		// Accept also a bare pass-through body as content if not provided
		// but here enforce content present for clarity
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: "empty content"})
	}
	if err := os.WriteFile(filepath.Join(base, file), req.Content, 0o644); err != nil {
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

// DeleteDiagram removes a diagram by name
func (h *Handlers) DeleteDiagram(c echo.Context) error {
	name := c.Param("name")
	base := cfg.ChariotConfig.DiagramPath
	if strings.TrimSpace(base) == "" {
		return c.JSON(http.StatusServiceUnavailable, ResultJSON{Result: "ERROR", Data: "diagram path not configured"})
	}
	file, err := sanitizeDiagramName(name)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	if err := os.Remove(filepath.Join(base, file)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return c.JSON(http.StatusNotFound, ResultJSON{Result: "ERROR", Data: "not found"})
		}
		return c.JSON(http.StatusInternalServerError, ResultJSON{Result: "ERROR", Data: err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}
