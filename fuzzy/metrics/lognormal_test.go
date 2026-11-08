package metrics

import (
	"fuzzylab/fuzzy/core"
	"fuzzylab/fuzzy/number"
	"math"
	"testing"
)

// Analytic integral of the truncated normalized PDF is independent of the
// alpha-grid quadrature; this catches errors in the model/metric integration.
func TestLogNormalAreaConvergence(t *testing.T) {
	for _, sigma := range []float64{0.2, 0.4, 0.7} {
		n, _ := number.NewLogNormal(0.5, sigma)
		s := n.Support()
		exact := ((n.CDF(s.Hi)-n.CDF(s.Lo))/n.Density(n.Mode()) - number.TailEpsilon*s.Width()) / (1 - number.TailEpsilon)
		coarse := Area(n, core.Config{AlphaLevels: 200})
		fine := Area(n, core.Config{AlphaLevels: 4000})
		if math.Abs(fine-exact) >= math.Abs(coarse-exact) {
			t.Fatalf("sigma=%g did not converge: exact=%g coarse=%g fine=%g", sigma, exact, coarse, fine)
		}
		if math.Abs(fine-exact)/exact > 0.002 {
			t.Fatalf("sigma=%g relative error too high: exact=%g fine=%g", sigma, exact, fine)
		}
	}
}
