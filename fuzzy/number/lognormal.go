package number

import (
	"fmt"
	"fuzzylab/fuzzy/core"
	"math"
)

// TailEpsilon is a relative density level, not a tail probability.
// The finite fuzzy model rescales max(0, f/f(mode)-epsilon) to height 1.
const TailEpsilon = 1e-4

// LogNormal stores parameters of ln X ~ N(Mu, Sigma²).
// Density and CDF describe the original distribution; Membership describes
// a compact approximation of its normalized density (see TailEpsilon).
type LogNormal struct{ Mu, Sigma float64 }

var _ core.Number = LogNormal{}

func NewLogNormal(mu, sigma float64) (LogNormal, error) {
	n := LogNormal{mu, sigma}
	if math.IsNaN(mu) || math.IsInf(mu, 0) || math.IsNaN(sigma) || math.IsInf(sigma, 0) || sigma <= 0 {
		return LogNormal{}, fmt.Errorf("%w: μ должен быть конечным, σ — конечным и > 0", core.ErrInvalidParams)
	}
	center := mu - sigma*sigma
	radius := sigma * math.Sqrt(-2*math.Log(TailEpsilon))
	// Conservative numerical limits keep products and moments representable.
	if math.IsInf(center, 0) || math.IsInf(radius, 0) || center-radius < -160 || center+radius > 160 {
		return LogNormal{}, fmt.Errorf("%w: параметры дают слишком большой численный диапазон; нужны μ−σ² ± σ√(−2 ln ε) в [−160,160]", core.ErrInvalidParams)
	}
	c := n.Support()
	if !(c.Lo < n.Mode() && n.Mode() < c.Hi) {
		return LogNormal{}, fmt.Errorf("%w: σ слишком мало для машинной точности", core.ErrInvalidParams)
	}
	return n, nil
}

func (n LogNormal) Mode() float64 { return math.Exp(n.Mu - n.Sigma*n.Sigma) }
func (n LogNormal) Density(x float64) float64 {
	if !(x > 0) || math.IsInf(x, 0) {
		return 0
	}
	z := (math.Log(x) - n.Mu) / n.Sigma
	return math.Exp(-0.5*z*z - math.Log(x) - math.Log(n.Sigma) - 0.5*math.Log(2*math.Pi))
}
func (n LogNormal) CDF(x float64) float64 {
	if !(x > 0) {
		return 0
	}
	return 0.5 * math.Erfc(-(math.Log(x)-n.Mu)/(n.Sigma*math.Sqrt2))
}

// RelativeDensity is f(x)/f(mode), computed without dividing small densities.
func (n LogNormal) RelativeDensity(x float64) float64 {
	if !(x > 0) || math.IsInf(x, 0) {
		return 0
	}
	z := (math.Log(x) - (n.Mu - n.Sigma*n.Sigma)) / n.Sigma
	return math.Exp(-0.5 * z * z)
}
func (n LogNormal) Membership(x float64) float64 {
	if !n.Support().Contains(x) {
		return 0
	}
	return math.Max(0, (n.RelativeDensity(x)-TailEpsilon)/(1-TailEpsilon))
}
func (n LogNormal) AlphaCut(alpha float64) (core.Interval, bool) {
	if math.IsNaN(alpha) || alpha < 0 || alpha > 1 {
		return core.Interval{}, false
	}
	if alpha == 1 {
		m := n.Mode()
		return core.Interval{Lo: m, Hi: m}, true
	}
	beta := TailEpsilon + (1-TailEpsilon)*alpha
	radius := n.Sigma * math.Sqrt(-2*math.Log(beta))
	center := n.Mu - n.Sigma*n.Sigma
	return core.Interval{Lo: math.Exp(center - radius), Hi: math.Exp(center + radius)}, true
}
func (n LogNormal) Support() core.Interval { c, _ := n.AlphaCut(0); return c }
func (n LogNormal) Height() float64        { return 1 }
func (n LogNormal) Kind() string           { return "lognormal" }
func (n LogNormal) String() string         { return fmt.Sprintf("LN(μ=%.4g, σ=%.4g)", n.Mu, n.Sigma) }
