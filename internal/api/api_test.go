package api

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tech-candidate-4343434/guestbook/internal/store"
)

func TestContentTemplateRenderEntriesXSS(t *testing.T) {
	entries := []store.Entry{
		{Name: "John", Message: "<script>alert('XSS')</script>"},
	}

	var buf bytes.Buffer
	if err := contentTmpl.Execute(&buf, entries); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, "<script>alert('XSS')</script>") {
		t.Errorf("contains XSS: %q", got)
	}
}

func TestContentTemplateEmptyState(t *testing.T) {
	var buf bytes.Buffer
	if err := contentTmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("execute template: %v", err)
	}

	if !strings.Contains(buf.String(), "No entries yet") {
		t.Error("rendered page is missing the empty state")
	}
}
