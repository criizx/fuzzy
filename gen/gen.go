// Package gen — генератор наборов логнормальных нечётких величин.
package gen

import (
	"fmt"
	"math"
	"math/rand"

	"fuzzylab/fuzzy/core"
	"fuzzylab/fuzzy/number"
)

// Range — отрезок допустимых значений параметра.
type Range struct{ Lo, Hi float64 }

// Params — общие параметры генерации: число величин, диапазоны μ и σ.
type Params struct {
	N         int
	Mu, Sigma Range
	Seed      int64
}

// CloudParams — параметры режима «облака»: K центров и шум как доля диапазона.
type CloudParams struct {
	Params
	K     int
	Noise float64
}

func (p Params) validate() error {
	for _, v := range []float64{p.Mu.Lo, p.Mu.Hi, p.Sigma.Lo, p.Sigma.Hi} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%w: границы диапазонов должны быть конечными", core.ErrInvalidParams)
		}
	}
	switch {
	case p.N < 1 || p.N > 300:
		return fmt.Errorf("%w: число величин N=%d (нужно от 1 до 300)", core.ErrInvalidParams, p.N)
	case p.Mu.Lo > p.Mu.Hi || p.Sigma.Lo > p.Sigma.Hi:
		return fmt.Errorf("%w: пустой диапазон (начало больше конца)", core.ErrInvalidParams)
	case !(p.Sigma.Lo > 0):
		return fmt.Errorf("%w: σ должна быть положительной", core.ErrInvalidParams)
	}
	return nil
}

func (r Range) draw(rnd *rand.Rand) float64 { return r.Lo + rnd.Float64()*(r.Hi-r.Lo) }

func (r Range) jitter(rnd *rand.Rand, noise float64) float64 {
	return (2*rnd.Float64() - 1) * noise * (r.Hi - r.Lo)
}

// Random строит N логнормальных величин с независимыми μ и σ, равномерными на заданных диапазонах.
func Random(p Params) ([]core.Number, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	rnd := rand.New(rand.NewSource(p.Seed))
	out := make([]core.Number, p.N)
	for i := range out {
		a := p.Mu.draw(rnd)
		u, err := number.NewLogNormal(a, p.Sigma.draw(rnd))
		if err != nil {
			return nil, err
		}
		out[i] = u
	}
	return out, nil
}

// Clouds строит N величин вокруг K случайных центров и возвращает номер центра каждой.
func Clouds(p CloudParams) ([]core.Number, []int, error) {
	if err := p.validate(); err != nil {
		return nil, nil, err
	}
	if p.K < 1 || p.K > p.N || p.Noise < 0 || math.IsNaN(p.Noise) || math.IsInf(p.Noise, 0) {
		return nil, nil, fmt.Errorf("%w: число облаков K=%d (нужно от 1 до N), шум=%g (нужно не меньше 0)", core.ErrInvalidParams, p.K, p.Noise)
	}
	rnd := rand.New(rand.NewSource(p.Seed))
	ca, cw := make([]float64, p.K), make([]float64, p.K)
	for k := range ca {
		ca[k], cw[k] = p.Mu.draw(rnd), p.Sigma.draw(rnd)
	}
	out := make([]core.Number, p.N)
	labels := make([]int, p.N)
	for i := range out {
		k := i % p.K
		a := min(max(ca[k]+p.Mu.jitter(rnd, p.Noise), p.Mu.Lo), p.Mu.Hi)
		w := min(max(cw[k]+p.Sigma.jitter(rnd, p.Noise), p.Sigma.Lo), p.Sigma.Hi)
		u, err := number.NewLogNormal(a, w)
		if err != nil {
			return nil, nil, err
		}
		out[i], labels[i] = u, k
	}
	return out, labels, nil
}
