package fuzzy

import (
	"errors"
	"math"
	"testing"
)

func TestLogNormalDensityAndCDF(t *testing.T) {
	n, err := NewLogNormal(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(n.Density(1)-1/math.Sqrt(2*math.Pi)) > 1e-14 {
		t.Fatal("density at 1")
	}
	if n.CDF(1) != 0.5 || n.CDF(0) != 0 || n.CDF(math.Inf(1)) != 1 {
		t.Fatal("CDF limits/median")
	}
	for _, x := range []float64{-1, 0, math.Inf(1), math.NaN()} {
		if n.Density(x) != 0 || n.Membership(x) != 0 {
			t.Fatalf("invalid x=%g", x)
		}
	}
	if math.Abs(n.Membership(n.Mode())-1) > 1e-14 {
		t.Fatal("peak must have membership 1")
	}
	for _, x := range []float64{0.1, 0.5, 1, 3} {
		if math.Abs(n.RelativeDensity(x)-n.Density(x)/n.Density(n.Mode())) > 1e-13 {
			t.Fatal("density normalization")
		}
	}
}

func TestLogNormalCutsMatchMembership(t *testing.T) {
	for _, params := range [][2]float64{{0, 0.3}, {-1, 0.7}, {2, 1.1}} {
		n, err := NewLogNormal(params[0], params[1])
		if err != nil {
			t.Fatal(err)
		}
		prev := n.Support()
		for _, a := range []float64{0, 1e-5, 0.01, 0.1, 0.5, 0.99, 1} {
			c, ok := n.AlphaCut(a)
			if !ok || c.Lo < prev.Lo || c.Hi > prev.Hi || c.Lo > c.Hi {
				t.Fatalf("non-nested cut at %g: %v", a, c)
			}
			for _, x := range []float64{c.Lo, c.Hi} {
				if math.Abs(n.Membership(x)-a) > 1e-12 {
					t.Fatalf("mu(%g)=%g want %g", x, n.Membership(x), a)
				}
			}
			if n.Membership(c.Mid())+1e-12 < a {
				t.Fatal("cut interior")
			}
			prev = c
		}
		if n.Support().Lo <= 0 || math.IsInf(n.Support().Hi, 0) {
			t.Fatal("finite positive support")
		}
		for _, a := range []float64{-0.1, 1.1, math.NaN()} {
			if _, ok := n.AlphaCut(a); ok {
				t.Fatal("invalid alpha accepted")
			}
		}
	}
}

func TestLogNormalRejectsInvalidParams(t *testing.T) {
	for _, p := range [][2]float64{{0, 0}, {0, -1}, {math.NaN(), 1}, {0, math.Inf(1)}, {1000, 1}, {0, 100}, {0, 1e-20}} {
		if _, err := NewLogNormal(p[0], p[1]); !errors.Is(err, ErrInvalidParams) {
			t.Fatalf("accepted %v: %v", p, err)
		}
	}
}
