package fuzzy

import (
	"fmt"
	"math"
)

// Uniform is a legacy right-shoulder fixture for regression tests.
// Its membership is the uniform CDF restricted to [A,B], not a rectangular density.
type Uniform struct{ A, B float64 }

var _ Number = Uniform{}

// NewUniform creates a reference value with finite parameters a < b.
func NewUniform(a, b float64) (Uniform, error) {
	if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) || !(a < b) {
		return Uniform{}, fmt.Errorf("%w: равномерная величина требует конечных a < b, получено a=%g b=%g",
			ErrInvalidParams, a, b)
	}
	return Uniform{A: a, B: b}, nil
}

// CDF returns the original uniform cumulative distribution function.
func (u Uniform) CDF(x float64) float64 {
	if math.IsNaN(x) || x <= u.A {
		return 0
	}
	if x >= u.B {
		return 1
	}
	return (x - u.A) / (u.B - u.A)
}

// Membership returns the uniform CDF on [A,B] and zero outside that support.
func (u Uniform) Membership(x float64) float64 {
	if math.IsNaN(x) || x < u.A || x > u.B {
		return 0
	}
	return u.CDF(x)
}

// Density returns 1/(B-A) inside [A,B] and zero outside.
func (u Uniform) Density(x float64) float64 {
	if x >= u.A && x <= u.B {
		return 1 / (u.B - u.A)
	}
	return 0
}

func (u Uniform) Support() Interval { return Interval{Lo: u.A, Hi: u.B} }

// AlphaCut returns [A+alpha*(B-A),B]; level zero denotes the support.
func (u Uniform) AlphaCut(alpha float64) (Interval, bool) {
	if math.IsNaN(alpha) || alpha < 0 || alpha > 1 {
		return Interval{}, false
	}
	if alpha == 0 {
		return u.Support(), true
	}
	if alpha == 1 {
		return Interval{Lo: u.B, Hi: u.B}, true
	}
	return Interval{Lo: u.A + alpha*(u.B-u.A), Hi: u.B}, true
}

func (u Uniform) Height() float64 { return 1 }
func (u Uniform) Kind() string    { return "uniform" }
func (u Uniform) Width() float64  { return u.B - u.A }
func (u Uniform) Mid() float64    { return (u.A + u.B) / 2 }

func (u Uniform) String() string { return fmt.Sprintf("U[%g; %g]", u.A, u.B) }
