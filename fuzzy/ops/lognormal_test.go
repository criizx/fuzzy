package ops

import (
	"fuzzylab/fuzzy/compare"
	"fuzzylab/fuzzy/core"
	"fuzzylab/fuzzy/number"
	"math"
	"testing"
)

func TestLogNormalArithmetic(t *testing.T) {
	a, _ := number.NewLogNormal(0, 0.3)
	b, _ := number.NewLogNormal(0.7, 0.4)
	for _, fn := range []func(core.Number, core.Number, core.Config) (number.Cuts, error){Add, Sub, Mul, Div} {
		r, err := fn(a, b, core.Config{})
		if err != nil {
			t.Fatal(err)
		}
		for i, c := range r.Intervals() {
			if !c.Valid() || math.IsInf(c.Lo, 0) || math.IsInf(c.Hi, 0) {
				t.Fatalf("invalid cut %d: %v", i, c)
			}
		}
	}
	r, _ := Add(a, b, core.Config{})
	c, _ := r.AlphaCut(1)
	if math.Abs(c.Lo-a.Mode()-b.Mode()) > 1e-12 || c.Lo != c.Hi {
		t.Fatal("sum of cores")
	}
	r, _ = Div(a, a, core.Config{})
	c, _ = r.AlphaCut(1)
	if c.Lo != 1 || c.Hi != 1 {
		t.Fatal("ratio core")
	}
	// Fuzzy interval arithmetic treats operands independently; A-A need not be crisp 0.
	r, _ = Sub(a, a, core.Config{})
	if r.Support().Width() <= 0 {
		t.Fatal("interval extension")
	}
	far, _ := number.NewLogNormal(4, 0.2)
	if compare.PossibilityGE(a, far) != 0 || compare.PossibilityGE(far, a) != 1 {
		t.Fatal("separated numbers comparison")
	}
}
