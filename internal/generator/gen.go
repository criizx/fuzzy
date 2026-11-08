// Package generator creates reproducible datasets of lognormal fuzzy values.
package generator

import (
	"fmt"
	"math"
	"math/rand"

	"fuzzylab/internal/fuzzy"
)

// Range specifies inclusive bounds for a generated parameter.
type Range struct{ Lo, Hi float64 }

// Params defines dataset size, Mu and Sigma ranges, and a reproducible random seed.
type Params struct {
	N         int
	Mu, Sigma Range
	Seed      int64
}

// CloudParams adds K centers and noise expressed as a fraction of each parameter range.
type CloudParams struct {
	Params
	K     int
	Noise float64
}

func (p Params) validate() error {
	for _, v := range []float64{p.Mu.Lo, p.Mu.Hi, p.Sigma.Lo, p.Sigma.Hi} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%w: границы диапазонов должны быть конечными", fuzzy.ErrInvalidParams)
		}
	}
	switch {
	case p.N < 1 || p.N > 300:
		return fmt.Errorf("%w: число величин N=%d (нужно от 1 до 300)", fuzzy.ErrInvalidParams, p.N)
	case p.Mu.Lo > p.Mu.Hi || p.Sigma.Lo > p.Sigma.Hi:
		return fmt.Errorf("%w: пустой диапазон (начало больше конца)", fuzzy.ErrInvalidParams)
	case !(p.Sigma.Lo > 0):
		return fmt.Errorf("%w: σ должна быть положительной", fuzzy.ErrInvalidParams)
	}
	return nil
}

func (r Range) draw(rnd *rand.Rand) float64 { return r.Lo + rnd.Float64()*(r.Hi-r.Lo) }

func (r Range) jitter(rnd *rand.Rand, noise float64) float64 {
	return (2*rnd.Float64() - 1) * noise * (r.Hi - r.Lo)
}

// Random creates N lognormal fuzzy values with independently sampled Mu and Sigma.
// Parameter sampling is uniform; the modeled variable X remains lognormal.
func Random(p Params) ([]fuzzy.Number, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	rnd := rand.New(rand.NewSource(p.Seed))
	out := make([]fuzzy.Number, p.N)
	for i := range out {
		a := p.Mu.draw(rnd)
		u, err := fuzzy.NewLogNormal(a, p.Sigma.draw(rnd))
		if err != nil {
			return nil, err
		}
		out[i] = u
	}
	return out, nil
}

// Clouds generates values around K random parameter centers and returns their center labels.
func Clouds(p CloudParams) ([]fuzzy.Number, []int, error) {
	if err := p.validate(); err != nil {
		return nil, nil, err
	}
	if p.K < 1 || p.K > p.N || p.Noise < 0 || math.IsNaN(p.Noise) || math.IsInf(p.Noise, 0) {
		return nil, nil, fmt.Errorf("%w: число облаков K=%d (нужно от 1 до N), шум=%g (нужно не меньше 0)", fuzzy.ErrInvalidParams, p.K, p.Noise)
	}
	rnd := rand.New(rand.NewSource(p.Seed))
	ca, cw := make([]float64, p.K), make([]float64, p.K)
	for k := range ca {
		ca[k], cw[k] = p.Mu.draw(rnd), p.Sigma.draw(rnd)
	}
	out := make([]fuzzy.Number, p.N)
	labels := make([]int, p.N)
	for i := range out {
		k := i % p.K
		a := min(max(ca[k]+p.Mu.jitter(rnd, p.Noise), p.Mu.Lo), p.Mu.Hi)
		w := min(max(cw[k]+p.Sigma.jitter(rnd, p.Noise), p.Sigma.Lo), p.Sigma.Hi)
		u, err := fuzzy.NewLogNormal(a, w)
		if err != nil {
			return nil, nil, err
		}
		out[i], labels[i] = u, k
	}
	return out, labels, nil
}
