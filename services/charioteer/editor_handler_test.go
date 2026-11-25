package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEditorIncludesBaselineFileScopes(t *testing.T) {
	recorder := httptest.NewRecorder()
	editorHandler(recorder, httptest.NewRequest("GET", "/charioteer/editor", nil))

	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	for _, scope := range []string{"global", "sandbox", "listeners", "bootstrap"} {
		if !strings.Contains(recorder.Body.String(), `<option value="`+scope+`">`) {
			t.Errorf("editor is missing %q Files scope", scope)
		}
	}
}

func TestEditorDraftAssetAndTemplate(t *testing.T) {
	for _, path := range []string{"/editor", "/charioteer/editor"} {
		recorder := httptest.NewRecorder()
		editorHandler(recorder, httptest.NewRequest("GET", path, nil))
		if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), `<script src="editor-state.js"></script>`) {
			t.Fatalf("%s: draft utility missing from rendered template", path)
		}
	}
	for _, path := range []string{"/editor-state.js", "/charioteer/editor-state.js"} {
		recorder := httptest.NewRecorder()
		editorStateJSHandler(recorder, httptest.NewRequest("GET", path, nil))
		if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "charioteer.editor.draft.v1:") {
			t.Fatalf("%s: embedded draft asset missing", path)
		}
		if got := recorder.Header().Get("Content-Type"); got != "application/javascript; charset=utf-8" {
			t.Fatalf("Content-Type = %q", got)
		}
	}
}
