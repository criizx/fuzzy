package fuzzy

import (
	"errors"
	"math"
	"testing"
)

func numberTestNear(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestNewUniformValidation(t *testing.T) {
	for _, p := range [][2]float64{{2, 2}, {3, 1}, {math.NaN(), 1}, {0, math.Inf(1)}} {
		if _, err := NewUniform(p[0], p[1]); !errors.Is(err, ErrInvalidParams) {
			t.Errorf("NewUniform(%v) err = %v", p, err)
		}
	}
}

func TestUniform(t *testing.T) {
	numberTestU, err := NewUniform(2, 6)
	if err != nil {
		t.Fatal(err)
	}

	// CDF исходного равномерного распределения.
	for _, tc := range []struct{ x, want float64 }{
		{1, 0}, {2, 0}, {3, 0.25}, {4, 0.5}, {5, 0.75}, {6, 1}, {7, 1},
	} {
		if got := numberTestU.CDF(tc.x); !numberTestNear(got, tc.want) {
			t.Errorf("CDF(%g) = %g, want %g", tc.x, got, tc.want)
		}
	}

	// Функция принадлежности — CDF, ограниченная исходным носителем.
	for _, tc := range []struct{ x, want float64 }{
		{1, 0}, {2, 0}, {3, 0.25}, {4, 0.5}, {5, 0.75}, {6, 1}, {7, 0},
	} {
		if got := numberTestU.Membership(tc.x); !numberTestNear(got, tc.want) {
			t.Errorf("Membership(%g) = %g, want %g", tc.x, got, tc.want)
		}
	}
	if numberTestU.Membership(math.NaN()) != 0 {
		t.Error("Membership(NaN) must be 0")
	}

	if !numberTestNear(numberTestU.Density(3), 0.25) || numberTestU.Density(7) != 0 {
		t.Error("density")
	}

	for _, tc := range []struct {
		alpha float64
		want  Interval
	}{
		{0, Interval{Lo: 2, Hi: 6}},
		{0.25, Interval{Lo: 3, Hi: 6}},
		{0.5, Interval{Lo: 4, Hi: 6}},
		{0.7, Interval{Lo: 4.8, Hi: 6}},
		{1, Interval{Lo: 6, Hi: 6}},
	} {
		got, ok := numberTestU.AlphaCut(tc.alpha)
		if !ok || !numberTestNear(got.Lo, tc.want.Lo) || !numberTestNear(got.Hi, tc.want.Hi) {
			t.Errorf("AlphaCut(%g) = %v, %v; want %v", tc.alpha, got, ok, tc.want)
		}
	}
	for _, alpha := range []float64{-0.1, 1.1, math.NaN()} {
		if _, ok := numberTestU.AlphaCut(alpha); ok {
			t.Errorf("AlphaCut(%g) must not exist", alpha)
		}
	}
	if numberTestU.Height() != 1 || numberTestU.Width() != 4 || numberTestU.Mid() != 4 {
		t.Error("height/width/mid")
	}
}

// треугольная величина — только для проверки общности Cuts
type numberTestTri struct{ a, m, b float64 }

func (t numberTestTri) Membership(x float64) float64 {
	switch {
	case x < t.a || x > t.b:
		return 0
	case x <= t.m:
		return (x - t.a) / (t.m - t.a)
	default:
		return (t.b - x) / (t.b - t.m)
	}
}
func (t numberTestTri) Support() Interval { return Interval{Lo: t.a, Hi: t.b} }
func (t numberTestTri) AlphaCut(al float64) (Interval, bool) {
	if al < 0 || al > 1 {
		return Interval{}, false
	}
	return Interval{Lo: t.a + al*(t.m-t.a), Hi: t.b - al*(t.b-t.m)}, true
}
func (t numberTestTri) Height() float64 { return 1 }
func (t numberTestTri) Kind() string    { return "numberTestTri" }

func TestCutsFromTriangle(t *testing.T) {
	c, err := FromNumber(numberTestTri{0, 2, 6}, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []float64{-1, 0, 0.5, 1, 2, 3, 4, 6, 7} {
		if got, want := c.Membership(x), (numberTestTri{0, 2, 6}).Membership(x); !numberTestNear(got, want) {
			t.Errorf("μ(%g) = %g, want %g", x, got, want)
		}
	}
	cut, ok := c.AlphaCut(0.5)
	if !ok || !numberTestNear(cut.Lo, 1) || !numberTestNear(cut.Hi, 4) {
		t.Errorf("cut(0.5) = %v %v", cut, ok)
	}
}

func TestCutsFromUniformCDF(t *testing.T) {
	numberTestU, _ := NewUniform(1, 3)
	c, err := FromNumber(numberTestU, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []float64{0.99, 1, 1.25, 1.5, 2, 2.75, 3, 3.01} {
		if got, want := c.Membership(x), numberTestU.Membership(x); !numberTestNear(got, want) {
			t.Errorf("Cuts.Membership(%g) = %g, want %g", x, got, want)
		}
	}
	cut, ok := c.AlphaCut(0.5)
	if !ok || !numberTestNear(cut.Lo, 2) || !numberTestNear(cut.Hi, 3) {
		t.Errorf("cut(0.5) = %v, %v", cut, ok)
	}
}

func TestNewCutsValidation(t *testing.T) {
	ok := []Interval{{Lo: 0, Hi: 4}, {Lo: 1, Hi: 3}}
	if _, err := NewCuts([]float64{0, 1}, ok); err != nil {
		t.Fatal(err)
	}
	bad := []struct {
		name string
		l    []float64
		c    []Interval
	}{
		{"empty", nil, nil},
		{"len mismatch", []float64{0, 1}, ok[:1]},
		{"not increasing", []float64{1, 0.5}, ok},
		{"not nested", []float64{0, 1}, []Interval{{Lo: 1, Hi: 3}, {Lo: 0, Hi: 4}}},
		{"level>1", []float64{0, 2}, ok},
		{"zero height", []float64{0}, ok[:1]},
	}
	for _, b := range bad {
		if _, err := NewCuts(b.l, b.c); !errors.Is(err, ErrInvalidParams) {
			t.Errorf("%s: err = %v", b.name, err)
		}
	}

	c, _ := NewCuts([]float64{0, 1}, ok)
	if _, valid := c.AlphaCut(-0.1); valid {
		t.Error("negative alpha cut must not exist")
	}
}

func TestCutsNormalized(t *testing.T) {
	c, _ := NewCuts([]float64{0, 0.5}, []Interval{{Lo: 0, Hi: 4}, {Lo: 1, Hi: 3}})
	n := c.Normalized()
	if n.Height() != 1 {
		t.Errorf("height = %g", n.Height())
	}
	if !numberTestNear(n.Membership(2), 1) || !numberTestNear(c.Membership(2), 0.5) {
		t.Error("normalization")
	}
}

func TestOutlineUniformCDF(t *testing.T) {
	numberTestU, _ := NewUniform(1, 3)
	pts := Outline(numberTestU, 10)
	if len(pts) < 4 || pts[0] != (Point{X: 1, Y: 0}) || pts[len(pts)-1] != (Point{X: 3, Y: 0}) {
		t.Errorf("outline endpoints: %v", pts)
	}
	if pts[10] != (Point{X: 3, Y: 1}) {
		t.Errorf("membership peak: %v", pts[10])
	}
	for i := 1; i <= 10; i++ {
		if pts[i].X < 1 || pts[i].X > 3 || pts[i].Y < pts[i-1].Y {
			t.Errorf("left branch is not increasing: %v", pts[:11])
		}
	}
	for i := 11; i < len(pts); i++ {
		if pts[i].X != 3 || pts[i].Y > pts[i-1].Y {
			t.Errorf("right vertical branch is invalid: %v", pts[i])
		}
	}
}
