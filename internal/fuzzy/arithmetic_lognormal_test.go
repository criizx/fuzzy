package fuzzy

import (
	"math"
	"testing"
)

func TestLogNormalArithmetic(t *testing.T) {
	a, _ := NewLogNormal(0, 0.3)
	b, _ := NewLogNormal(0.7, 0.4)
	for _, fn := range []func(Number, Number, Config) (Cuts, error){Add, Sub, Mul, Div} {
		r, err := fn(a, b, Config{})
		if err != nil {
			t.Fatal(err)
		}
		for i, c := range r.Intervals() {
			if !c.Valid() || math.IsInf(c.Lo, 0) || math.IsInf(c.Hi, 0) {
				t.Fatalf("invalid cut %d: %v", i, c)
			}
		}
	}
	r, _ := Add(a, b, Config{})
	c, _ := r.AlphaCut(1)
	if math.Abs(c.Lo-a.Mode()-b.Mode()) > 1e-12 || c.Lo != c.Hi {
		t.Fatal("sum of cores")
	}
	r, _ = Div(a, a, Config{})
	c, _ = r.AlphaCut(1)
	if c.Lo != 1 || c.Hi != 1 {
		t.Fatal("ratio core")
	}
	// Fuzzy interval arithmetic treats operands independently; A-A need not be crisp 0.
	r, _ = Sub(a, a, Config{})
	if r.Support().Width() <= 0 {
		t.Fatal("interval extension")
	}
	far, _ := NewLogNormal(4, 0.2)
	if PossibilityGE(a, far) != 0 || PossibilityGE(far, a) != 1 {
		t.Fatal("separated numbers comparison")
	}
}
