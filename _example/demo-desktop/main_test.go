package main

import "testing"

func TestNormalizeExternalBrowserURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "http", value: "http://example.com/a?b=c", want: "http://example.com/a?b=c"},
		{name: "https uppercase scheme", value: "HTTPS://example.com/a%20b", want: "https://example.com/a%20b"},
		{name: "empty", value: "", wantErr: true},
		{name: "relative", value: "/docs", wantErr: true},
		{name: "missing host", value: "https:///docs", wantErr: true},
		{name: "javascript", value: "javascript:alert(1)", wantErr: true},
		{name: "mailto", value: "mailto:user@example.com", wantErr: true},
		{name: "raw space", value: "https://example.com/a b", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeExternalBrowserURL(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
