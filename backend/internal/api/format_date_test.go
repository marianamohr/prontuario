package api

import "testing"

func TestFormatDateBR(t *testing.T) {
	if got := formatDateBR("2026-02-11"); got != "11/02/2026" {
		t.Fatalf("expected 11/02/2026, got %q", got)
	}
	if got := formatDateBR(""); got != "" {
		t.Fatalf("expected empty for empty input, got %q", got)
	}
	if got := formatDateBR("invalid"); got != "" {
		t.Fatalf("expected empty for invalid input, got %q", got)
	}
}

