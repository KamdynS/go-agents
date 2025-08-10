package tools

import (
	"context"
	"testing"
)

func TestCalculatorBasicOps(t *testing.T) {
	c := &CalculatorTool{}
	tests := []struct{ in, want string }{
		{"add 1 2", "3"},
		{"sub 5 2", "3"},
		{"mul 3 4", "12"},
		{"div 8 2", "4"},
		{"pow 2 3", "8"},
		{"sqrt 9", "3"},
	}
	for _, tc := range tests {
		got, err := c.Execute(context.Background(), tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("%s => %q (%v), want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestCalculatorErrors(t *testing.T) {
	c := &CalculatorTool{}
	cases := []string{"", "x", "add 1", "div 1 0", "sqrt -1", "noop 1 2"}
	for _, in := range cases {
		if _, err := c.Execute(context.Background(), in); err == nil {
			t.Fatalf("expected error for %q", in)
		}
	}
}
