package fuzzy

import (
	"errors"
	"math"
	"testing"
)

func opsTestNear(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func opsTestU(t *testing.T, a, b float64) Uniform {
	t.Helper()
	x, err := NewUniform(a, b)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func checkSupport(t *testing.T, name string, n Number, lo, hi float64) {
	t.Helper()
	s := n.Support()
	if !opsTestNear(s.Lo, lo) || !opsTestNear(s.Hi, hi) {
		t.Errorf("%s: support = %v, want [%g, %g]", name, s, lo, hi)
	}
}

func TestUniformCDFArithmetic(t *testing.T) {
	cfg := Config{}
	a, b := opsTestU(t, 1, 3), opsTestU(t, 2, 5)

	r, err := Add(a, b, cfg)
	if err != nil {
		t.Fatal(err)
	}
	checkSupport(t, "Add", r, 3, 8)

	r, _ = Sub(a, b, cfg)
	checkSupport(t, "Sub", r, -4, 1)

	r, _ = Mul(a, b, cfg)
	checkSupport(t, "Mul", r, 2, 15)

	r, err = Div(a, b, cfg)
	if err != nil {
		t.Fatal(err)
	}
	checkSupport(t, "Div", r, 0.2, 1.5)

	r, _ = Neg(a, cfg)
	checkSupport(t, "Neg", r, -3, -1)

	r, _ = Scale(a, -2, cfg)
	checkSupport(t, "Scale", r, -6, -2)
}

func TestUniformCDFAdditionAlphaCuts(t *testing.T) {
	a, b := opsTestU(t, 1, 3), opsTestU(t, 2, 5)
	sum, err := Add(a, b, Config{})
	if err != nil {
		t.Fatal(err)
	}
	cut, ok := sum.AlphaCut(0.5)
	if !ok || !opsTestNear(cut.Lo, 5.5) || !opsTestNear(cut.Hi, 8) {
		t.Errorf("sum α-cut at 0.5 = %v, %v; want [5.5, 8]", cut, ok)
	}
	if !opsTestNear(sum.Membership(5.5), 0.5) || !opsTestNear(sum.Membership(8), 1) {
		t.Errorf("unexpected sum membership at left boundary or peak")
	}
}

func TestDivByZeroSupport(t *testing.T) {
	_, err := Div(opsTestU(t, 1, 2), opsTestU(t, -1, 1), Config{})
	if !errors.Is(err, ErrDivByZero) {
		t.Errorf("err = %v", err)
	}
}

type opsTestTri struct{ a, m, b float64 }

func (t opsTestTri) Membership(x float64) float64 {
	switch {
	case x < t.a || x > t.b:
		return 0
	case x <= t.m:
		return (x - t.a) / (t.m - t.a)
	default:
		return (t.b - x) / (t.b - t.m)
	}
}
func (t opsTestTri) Support() Interval { return Interval{Lo: t.a, Hi: t.b} }
func (t opsTestTri) AlphaCut(al float64) (Interval, bool) {
	if al < 0 || al > 1 {
		return Interval{}, false
	}
	return Interval{Lo: t.a + al*(t.m-t.a), Hi: t.b - al*(t.b-t.m)}, true
}
func (t opsTestTri) Height() float64 { return 1 }
func (t opsTestTri) Kind() string    { return "opsTestTri" }

// Сумма треугольных (0,1,2) и (1,2,3) — треугольная (1,3,5).
func TestTriangleSum(t *testing.T) {
	r, err := Add(opsTestTri{0, 1, 2}, opsTestTri{1, 2, 3}, Config{})
	if err != nil {
		t.Fatal(err)
	}
	want := opsTestTri{1, 3, 5}
	for _, x := range []float64{0.5, 1, 1.5, 2, 2.5, 3, 4, 5, 5.5} {
		if got := r.Membership(x); !opsTestNear(got, want.Membership(x)) {
			t.Errorf("μ(%g) = %g, want %g", x, got, want.Membership(x))
		}
	}
}

// Цепочка: результат операции снова является Number.
func TestChain(t *testing.T) {
	calc := Calculator{}
	s, err := calc.Add(opsTestU(t, 1, 2), opsTestU(t, 3, 4)) // [4,6]
	if err != nil {
		t.Fatal(err)
	}
	p, err := calc.Mul(s, opsTestU(t, 2, 3)) // [8,18]
	if err != nil {
		t.Fatal(err)
	}
	checkSupport(t, "chain", p, 8, 18)
}

func TestCustomOperation(t *testing.T) {
	r, err := Calculator{}.Apply(opsTestU(t, 1, 5), opsTestU(t, 3, 4), MaxOp)
	if err != nil {
		t.Fatal(err)
	}
	checkSupport(t, "Max", r, 3, 5)
}
