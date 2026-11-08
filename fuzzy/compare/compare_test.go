package compare

import (
	"errors"
	"math"
	"testing"

	"fuzzylab/fuzzy/core"
	"fuzzylab/fuzzy/number"
)

func u(t *testing.T, a, b float64) core.Number {
	t.Helper()
	x, err := number.NewUniform(a, b)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func TestPossibilityNecessity(t *testing.T) {
	lo, hi, mid := u(t, 1, 2), u(t, 5, 6), u(t, 1.5, 5.5)

	if v := PossibilityGE(hi, lo); v != 1 {
		t.Errorf("Pos(hi>=lo) = %g", v)
	}
	if v := PossibilityGE(lo, hi); v != 0 {
		t.Errorf("Pos(lo>=hi) = %g", v)
	}
	if v := NecessityGE(hi, lo); v != 1 {
		t.Errorf("Nec(hi>=lo) = %g", v)
	}
	if v := NecessityGE(lo, hi); v != 0 {
		t.Errorf("Nec(lo>=hi) = %g", v)
	}
	// Для правосторонних треугольников перекрытие не означает нулевую
	// необходимость: Pos(lo >= mid) = (2-1.5)/(5.5-1.5) = 0.125.
	if got := PossibilityGE(mid, lo); math.Abs(got-1) > 1e-9 {
		t.Errorf("Pos(mid>=lo) = %g, want 1", got)
	}
	if got := PossibilityGE(lo, mid); math.Abs(got-0.125) > 1e-9 {
		t.Errorf("Pos(lo>=mid) = %g, want 0.125", got)
	}
	if got := NecessityGE(mid, lo); math.Abs(got-0.875) > 1e-9 {
		t.Errorf("Nec(mid>=lo) = %g, want 0.875", got)
	}
}

func TestStrategies(t *testing.T) {
	a, b := u(t, 1, 2), u(t, 5, 6)
	for _, s := range []Strategy{
		CentroidStrategy{}, PossibilityStrategy{}, IntervalStrategy{Alpha: 0.5},
	} {
		if s.Compare(a, b) != Less || s.Compare(b, a) != Greater || s.Compare(a, a) != Equal {
			t.Errorf("%s: wrong order", s.Name())
		}
	}
}

func TestSortMaxMin(t *testing.T) {
	items := []core.Number{u(t, 5, 6), u(t, 1, 2), u(t, 3, 4)}
	s := CentroidStrategy{}

	mx, err := Max(items, s)
	if err != nil || mx.Support().Lo != 5 {
		t.Errorf("Max = %v, %v", mx, err)
	}
	mn, _ := Min(items, s)
	if mn.Support().Lo != 1 {
		t.Errorf("Min = %v", mn)
	}

	Sort(items, s)
	for i, want := range []float64{1, 3, 5} {
		if got := items[i].Support().Lo; math.Abs(got-want) > 1e-12 {
			t.Errorf("items[%d].Lo = %g, want %g", i, got, want)
		}
	}

	if _, err := Max(nil, s); !errors.Is(err, core.ErrEmpty) {
		t.Errorf("empty Max err = %v", err)
	}
}

func crisp(t *testing.T, x float64) core.Number {
	t.Helper()
	c, err := number.NewCuts([]float64{0, 1}, []core.Interval{{Lo: x, Hi: x}, {Lo: x, Hi: x}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestStrictPossibilityAndNecessity(t *testing.T) {
	zero, one := crisp(t, 0), crisp(t, 1)
	cases := []struct {
		name    string
		a, b    core.Number
		gt, nec float64
	}{
		{"identical crisp", zero, zero, 0, 1},
		{"ordered crisp", one, zero, 1, 1},
		{"reverse crisp", zero, one, 0, 0},
		{"touching supports", u(t, 0, 1), u(t, 1, 2), 0, 0},
		{"reverse touching supports", u(t, 1, 2), u(t, 0, 1), 1, 1},
		{"identical continuous", u(t, 0, 1), u(t, 0, 1), 1, 0},
		{"overlap", u(t, 1, 2), u(t, 1.5, 5.5), 0.125, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := PossibilityGT(c.a, c.b); math.Abs(got-c.gt) > 1e-12 {
				t.Errorf("PosGT=%g want %g", got, c.gt)
			}
			if got := NecessityGE(c.a, c.b); math.Abs(got-c.nec) > 1e-12 {
				t.Errorf("NecGE=%g want %g", got, c.nec)
			}
		})
	}
	n, _ := number.NewLogNormal(0, 0.3)
	if got := PossibilityGT(n, n); math.Abs(got-1) > 1e-12 {
		t.Fatalf("coincident lognormal modes: strict supremum=%g want 1", got)
	}
}
