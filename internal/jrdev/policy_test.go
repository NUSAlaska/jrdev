package jrdev

import "testing"

func TestDefaultMaxIterations(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{0, 3},
		{1, 5},
		{2, 7},
		{10, 23},
	}
	for _, tt := range tests {
		if g := DefaultMaxIterations(tt.n); g != tt.want {
			t.Errorf("DefaultMaxIterations(%d) = %d; want %d", tt.n, g, tt.want)
		}
	}
}

func TestEffectiveMaxIterations(t *testing.T) {
	if g := EffectiveMaxIterations(5, 0); g != 13 {
		t.Fatalf("override 0: got %d", g)
	}
	if g := EffectiveMaxIterations(5, 100); g != 100 {
		t.Fatalf("override 100: got %d", g)
	}
}
