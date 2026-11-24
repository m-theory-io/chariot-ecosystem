package main

import (
	_ "embed"
	"net/http"
)

//go:embed templates/editor-state.js
var editorStateJS []byte

func editorStateJSHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(editorStateJS)
}
