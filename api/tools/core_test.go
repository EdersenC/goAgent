package tools

import "testing"

func TestParsePageNumber(t *testing.T) {
	tests := []struct {
		name    string
		raw     interface{}
		want    int
		wantErr bool
	}{
		{name: "empty defaults to one", raw: "", want: 1},
		{name: "valid number", raw: "3", want: 3},
		{name: "invalid string", raw: "abc", wantErr: true},
		{name: "invalid type", raw: 2, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePageNumber(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNormalizeQueries(t *testing.T) {
	tests := []struct {
		name    string
		raw     interface{}
		want    []string
		wantErr bool
	}{
		{name: "slice interface", raw: []interface{}{"a", "b"}, want: []string{"a", "b"}},
		{name: "slice string", raw: []string{"a", "b"}, want: []string{"a", "b"}},
		{name: "json list string", raw: `["a","b"]`, want: []string{"a", "b"}},
		{name: "single quoted list", raw: `['a','b']`, want: []string{"a", "b"}},
		{name: "stringified json list", raw: `"[\"a\",\"b\"]"`, want: []string{"a", "b"}},
		{name: "invalid string", raw: `abc`, wantErr: true},
		{name: "invalid type", raw: 10, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeQueries(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got len=%d, want len=%d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got[%d]=%q, want[%d]=%q", i, got[i], i, tt.want[i])
				}
			}
		})
	}
}
