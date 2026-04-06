package logger

import (
	"strings"
	"testing"
)

func TestRedactURLSecrets(t *testing.T) {
	in := "https://example.com/rest/stream?u=user&t=abc123&s=salt&id=42"
	out := RedactURLSecrets(in)

	if out == in {
		t.Fatalf("expected redacted output, got original URL")
	}
	if strings.Contains(out, "u=user") || strings.Contains(out, "t=abc123") || strings.Contains(out, "s=salt") {
		t.Fatalf("expected sensitive query params to be redacted, got: %s", out)
	}
}
