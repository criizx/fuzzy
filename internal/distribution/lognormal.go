// Package distribution implements probability distributions independently of fuzzy models.
package distribution

import (
	"fmt"
	"math"
)

// LogNormal describes ln X ~ N(Mu, Sigma²). Its support is (0, +Inf).
type LogNormal struct{ Mu, Sigma float64 }

// NewLogNormal validates finite parameters and a strictly positive scale.
func NewLogNormal(mu, sigma float64) (LogNormal, error) {
	if math.IsNaN(mu) || math.IsInf(mu, 0) || math.IsNaN(sigma) || math.IsInf(sigma, 0) || sigma <= 0 {
		return LogNormal{}, fmt.Errorf("μ должен быть конечным, σ — конечным и > 0")
	}
	return LogNormal{Mu: mu, Sigma: sigma}, nil
}

// Mode returns the location of the maximum probability density.
func (n LogNormal) Mode() float64 { return math.Exp(n.Mu - n.Sigma*n.Sigma) }

// Median returns the median of X, not of ln X.
func (n LogNormal) Median() float64 { return math.Exp(n.Mu) }

// Mean returns the probabilistic expectation of X.
func (n LogNormal) Mean() float64 { return math.Exp(n.Mu + n.Sigma*n.Sigma/2) }

// Variance returns the probabilistic variance of X.
func (n LogNormal) Variance() float64 {
	s2 := n.Sigma * n.Sigma
	return math.Expm1(s2) * math.Exp(2*n.Mu+s2)
}

// Density returns the probability density, which need not be bounded by one.
func (n LogNormal) Density(x float64) float64 {
	if !(x > 0) || math.IsInf(x, 0) {
		return 0
	}
	z := (math.Log(x) - n.Mu) / n.Sigma
	return math.Exp(-0.5*z*z - math.Log(x) - math.Log(n.Sigma) - 0.5*math.Log(2*math.Pi))
}

// CDF returns P(X <= x). As elsewhere in this project, NaN input returns zero.
func (n LogNormal) CDF(x float64) float64 {
	if !(x > 0) {
		return 0
	}
	return 0.5 * math.Erfc(-(math.Log(x)-n.Mu)/(n.Sigma*math.Sqrt2))
}

// RelativeDensity returns f(x)/f(mode) without dividing small densities.
func (n LogNormal) RelativeDensity(x float64) float64 {
	if !(x > 0) || math.IsInf(x, 0) {
		return 0
	}
	z := (math.Log(x) - (n.Mu - n.Sigma*n.Sigma)) / n.Sigma
	return math.Exp(-0.5 * z * z)
}
