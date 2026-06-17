package export

import (
	"encoding/json"
	"testing"
)

func TestNormalizeIDAcceptsIntegralValues(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want int
	}{
		{name: "int", in: 42, want: 42},
		{name: "int64", in: int64(42), want: 42},
		{name: "uint64", in: uint64(42), want: 42},
		{name: "float64 integer", in: float64(42), want: 42},
		{name: "string integer", in: "42", want: 42},
		{name: "string integer float", in: "42.0", want: 42},
		{name: "json number", in: json.Number("42"), want: 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizeID(tt.in)
			if !ok {
				t.Fatalf("NormalizeID(%v) rejected", tt.in)
			}
			if got != tt.want {
				t.Fatalf("NormalizeID(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeIDRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		in   any
	}{
		{name: "fractional float", in: 42.5},
		{name: "fractional string", in: "42.5"},
		{name: "text", in: "not-an-id"},
		{name: "empty string", in: ""},
		{name: "json fractional number", in: json.Number("42.5")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := NormalizeID(tt.in); ok {
				t.Fatalf("NormalizeID(%v) = %d, want rejected", tt.in, got)
			}
		})
	}
}
