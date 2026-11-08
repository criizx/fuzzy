package distribution

import (
	"math"
	"testing"
)

func TestLogNormalAnalyticValues(t *testing.T) {
	n, err := NewLogNormal(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		got, want float64
	}{
		{"mode", n.Mode(), math.Exp(-1)},
		{"median", n.Median(), 1},
		{"mean", n.Mean(), math.Exp(0.5)},
		{"variance", n.Variance(), (math.E - 1) * math.E},
		{"density at one", n.Density(1), 1 / math.Sqrt(2*math.Pi)},
		{"CDF at median", n.CDF(1), 0.5},
		{"CDF one log standard deviation", n.CDF(math.E), 0.8413447460685429},
		{"CDF positive infinity", n.CDF(math.Inf(1)), 1},
	} {
		if math.Abs(tc.got-tc.want) > 1e-12 {
			t.Errorf("%s: got %g, want %g", tc.name, tc.got, tc.want)
		}
	}
	for _, x := range []float64{-1, 0, math.NaN(), math.Inf(-1)} {
		if n.Density(x) != 0 || n.CDF(x) != 0 {
			t.Errorf("outside support: x=%g", x)
		}
	}
	if n.Density(math.Inf(1)) != 0 {
		t.Fatal("density at +Inf must vanish")
	}
}

func TestLogNormalProbabilityMass(t *testing.T) {
	n, _ := NewLogNormal(0.7, 0.4)
	const steps = 20000
	total := 0.0
	dt := 20.0 / steps
	for i := 0; i < steps; i++ {
		z := -10 + (float64(i)+0.5)*dt
		x := math.Exp(n.Mu + n.Sigma*z)
		total += n.Density(x) * x * n.Sigma * dt
	}
	if math.Abs(total-1) > 1e-12 {
		t.Fatalf("integral=%g, want 1", total)
	}
}

func TestLogNormalRejectsInvalidParameters(t *testing.T) {
	for _, p := range [][2]float64{{0, 0}, {0, -1}, {math.NaN(), 1}, {0, math.NaN()}, {math.Inf(1), 1}, {0, math.Inf(1)}} {
		if _, err := NewLogNormal(p[0], p[1]); err == nil {
			t.Errorf("accepted %v", p)
		}
	}
}

func TestProbabilityDensityIsNotMembership(t *testing.T) {
	n, _ := NewLogNormal(0, 0.1)
	if n.Density(n.Mode()) <= 1 || n.RelativeDensity(n.Mode()) != 1 {
		t.Fatal("probability density was incorrectly clipped or normalized")
	}
}
