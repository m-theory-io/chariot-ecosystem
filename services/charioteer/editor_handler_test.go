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
