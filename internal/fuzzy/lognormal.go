package fuzzy

import (
	"fmt"
	"math"

	"fuzzylab/internal/distribution"
)

// TailEpsilon is a relative density level, not a tail probability.
// The finite fuzzy model rescales max(0, f/f(mode)-epsilon) to height 1.
const TailEpsilon = 1e-4

// LogNormal stores parameters of ln X ~ N(Mu, Sigma²).
// Density and CDF describe the original distribution; Membership describes
// a compact approximation of its normalized density (see TailEpsilon).
type LogNormal struct{ Mu, Sigma float64 }

var _ Number = LogNormal{}

func NewLogNormal(mu, sigma float64) (LogNormal, error) {
	n := LogNormal{mu, sigma}
	if _, err := distribution.NewLogNormal(mu, sigma); err != nil {
		return LogNormal{}, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}
	center := mu - sigma*sigma
	radius := sigma * math.Sqrt(-2*math.Log(TailEpsilon))
	// Conservative numerical limits keep products and moments representable.
	if math.IsInf(center, 0) || math.IsInf(radius, 0) || center-radius < -160 || center+radius > 160 {
		return LogNormal{}, fmt.Errorf("%w: параметры дают слишком большой численный диапазон; нужны μ−σ² ± σ√(−2 ln ε) в [−160,160]", ErrInvalidParams)
	}
	c := n.Support()
	if !(c.Lo < n.Mode() && n.Mode() < c.Hi) {
		return LogNormal{}, fmt.Errorf("%w: σ слишком мало для машинной точности", ErrInvalidParams)
	}
	return n, nil
}

// Distribution returns the untruncated probability model underlying this fuzzy number.
func (n LogNormal) Distribution() distribution.LogNormal {
	return distribution.LogNormal{Mu: n.Mu, Sigma: n.Sigma}
}

// Mode returns the mode shared by the density and the membership function.
func (n LogNormal) Mode() float64 { return n.Distribution().Mode() }

// Density delegates to the original, untruncated probability density.
func (n LogNormal) Density(x float64) float64 { return n.Distribution().Density(x) }

// CDF delegates to the original probability distribution.
func (n LogNormal) CDF(x float64) float64 { return n.Distribution().CDF(x) }

// RelativeDensity returns f(x)/f(mode), before the fuzzy tail cutoff.
func (n LogNormal) RelativeDensity(x float64) float64 { return n.Distribution().RelativeDensity(x) }

func (n LogNormal) Membership(x float64) float64 {
	if !n.Support().Contains(x) {
		return 0
	}
	return math.Max(0, (n.RelativeDensity(x)-TailEpsilon)/(1-TailEpsilon))
}
func (n LogNormal) AlphaCut(alpha float64) (Interval, bool) {
	if math.IsNaN(alpha) || alpha < 0 || alpha > 1 {
		return Interval{}, false
	}
	if alpha == 1 {
		m := n.Mode()
		return Interval{Lo: m, Hi: m}, true
	}
	beta := TailEpsilon + (1-TailEpsilon)*alpha
	radius := n.Sigma * math.Sqrt(-2*math.Log(beta))
	center := n.Mu - n.Sigma*n.Sigma
	return Interval{Lo: math.Exp(center - radius), Hi: math.Exp(center + radius)}, true
}
func (n LogNormal) Support() Interval { c, _ := n.AlphaCut(0); return c }
func (n LogNormal) Height() float64   { return 1 }
func (n LogNormal) Kind() string      { return "lognormal" }
func (n LogNormal) String() string    { return fmt.Sprintf("LN(μ=%.4g, σ=%.4g)", n.Mu, n.Sigma) }
