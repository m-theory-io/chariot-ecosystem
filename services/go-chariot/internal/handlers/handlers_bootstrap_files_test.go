package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/labstack/echo/v4"
)

func withBootstrapFileConfig(t *testing.T, enabled bool) string {
	t.Helper()
	originalDataPath := cfg.ChariotConfig.DataPath
	originalBootstrap := cfg.ChariotConfig.Bootstrap
	originalEnabled := cfg.ChariotConfig.BootstrapEditEnabled
	dataPath := t.TempDir()
	cfg.ChariotConfig.DataPath = dataPath
	cfg.ChariotConfig.Bootstrap = "bootstrap.ch"
	cfg.ChariotConfig.BootstrapEditEnabled = enabled
	t.Cleanup(func() {
		cfg.ChariotConfig.DataPath = originalDataPath
		cfg.ChariotConfig.Bootstrap = originalBootstrap
		cfg.ChariotConfig.BootstrapEditEnabled = originalEnabled
	})
	return dataPath
}

func newBootstrapFileContext(method, target, body string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("session", &chariot.Session{UserID: "test-user", Username: "test-user"})
	return c, rec
}

func decodeBootstrapResult(t *testing.T, rec *httptest.ResponseRecorder) ResultJSON {
	t.Helper()
	var result ResultJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func TestBootstrapScopeRequiresExplicitEnablement(t *testing.T) {
	withBootstrapFileConfig(t, false)
	h := &Handlers{}
	c, rec := newBootstrapFileContext(http.MethodGet, "/api/files?scope=bootstrap", "")

	if err := h.ListFiles(c); err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	result := decodeBootstrapResult(t, rec)
	if result.Result != "ERROR" {
		t.Fatalf("expected error result, got %#v", result)
	}
}

func TestBootstrapScopeListGetSaveConfiguredBootstrap(t *testing.T) {
	dataPath := withBootstrapFileConfig(t, true)
	if err := os.WriteFile(filepath.Join(dataPath, "bootstrap.ch"), []byte("setq(x, 1)"), 0o644); err != nil {
		t.Fatalf("write bootstrap fixture: %v", err)
	}
	h := &Handlers{}

	c, rec := newBootstrapFileContext(http.MethodGet, "/api/files?scope=bootstrap", "")
	if err := h.ListFiles(c); err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected list status %d, got %d", http.StatusOK, rec.Code)
	}
	result := decodeBootstrapResult(t, rec)
	files, ok := result.Data.([]any)
	if !ok || len(files) != 1 || files[0] != "bootstrap.ch" {
		t.Fatalf("expected one bootstrap file, got %#v", result.Data)
	}

	c, rec = newBootstrapFileContext(http.MethodGet, "/api/files/bootstrap.ch?scope=bootstrap", "")
	c.SetParamNames("name")
	c.SetParamValues("bootstrap.ch")
	if err := h.GetFile(c); err != nil {
		t.Fatalf("GetFile returned error: %v", err)
	}
	result = decodeBootstrapResult(t, rec)
	if result.Data != "setq(x, 1)" {
		t.Fatalf("expected bootstrap content, got %#v", result.Data)
	}

	c, rec = newBootstrapFileContext(http.MethodPost, "/api/files?scope=bootstrap", `{"name":"bootstrap.ch","content":"setq(x, 2)"}`)
	if err := h.SaveFile(c); err != nil {
		t.Fatalf("SaveFile returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected save status %d, got %d", http.StatusOK, rec.Code)
	}
	content, err := os.ReadFile(filepath.Join(dataPath, "bootstrap.ch"))
	if err != nil {
		t.Fatalf("read saved bootstrap: %v", err)
	}
	if string(content) != "setq(x, 2)" {
		t.Fatalf("expected saved content, got %q", string(content))
	}
}

func TestBootstrapScopeRejectsOtherFiles(t *testing.T) {
	withBootstrapFileConfig(t, true)
	h := &Handlers{}
	c, rec := newBootstrapFileContext(http.MethodGet, "/api/files/other.ch?scope=bootstrap", "")
	c.SetParamNames("name")
	c.SetParamValues("other.ch")

	if err := h.GetFile(c); err != nil {
		t.Fatalf("GetFile returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
