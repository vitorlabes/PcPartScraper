package scraper

import (
	"testing"
)

func TestParsePrice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "preço normal",
			input:    "R$ 1.234,56",
			expected: 1234.56,
		},
		{
			name:     "preço com caractere especial",
			input:    "R$Â 941,16",
			expected: 941.16,
		},
		{
			name:     "preço sem centavos",
			input:    "R$ 1.000",
			expected: 1000.00,
		},
		{
			name:     "preço com espaços extras",
			input:    "  R$ 599,99  ",
			expected: 599.99,
		},
		{
			name:     "preço inválido",
			input:    "R$ abc",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePrice(tt.input)
			if got != tt.expected {
				t.Errorf("parsePrice(%q) = %f, want %f", tt.input, got, tt.expected)
			}
		})
	}
}
