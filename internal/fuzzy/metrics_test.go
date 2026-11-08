package fuzzy

import (
	"math"
	"testing"
)

func metricsTestNear(a, b, tol float64) bool { return math.Abs(a-b) < tol }

func metricsTestU(t *testing.T, a, b float64) Uniform {
	t.Helper()
	x, err := NewUniform(a, b)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func TestCharacteristicsUniformCDF(t *testing.T) {
	x := metricsTestU(t, 2, 6)
	cfg := Config{}
	if v := Area(x, cfg); !metricsTestNear(v, 2, 1e-9) { // (b-a)/2
		t.Errorf("Area = %g", v)
	}
	// Centroid, Mean и Variance интегрируются методом трапеций по 50 уровням;
	// допуск учитывает малую погрешность численного интегрирования.
	if v := Centroid(x, cfg); !metricsTestNear(v, 14.0/3.0, 5e-4) { // (a+2b)/3
		t.Errorf("Centroid = %g", v)
	}
	if v := Mean(x, cfg); !metricsTestNear(v, 32.0/6.0, 5e-4) { // (a+5b)/6
		t.Errorf("Mean = %g", v)
	}
	if v := Variance(x, cfg); !metricsTestNear(v, 2.0/3.0, 5e-4) { // (b-a)²/24
		t.Errorf("Variance = %g", v)
	}
	if v := Spread(x); v != 4 {
		t.Errorf("Spread = %g", v)
	}
}

func TestDistancesUniformCDF(t *testing.T) {
	a, b := metricsTestU(t, 0, 2), metricsTestU(t, 3, 7) // ΔLo(α)=3+2α, ΔHi(α)=5
	cfg := Config{}

	if v := (CutsL1Distance{Cfg: cfg}).Between(a, b); !metricsTestNear(v, 4.5, 1e-9) {
		t.Errorf("L1 = %g", v)
	}
	if v := (CutsL2Distance{Cfg: cfg}).Between(a, b); !metricsTestNear(v, math.Sqrt(62.0/3.0), 1e-4) {
		t.Errorf("L2 = %g", v)
	}
	if v := (HausdorffDistance{Cfg: cfg}).Between(a, b); !metricsTestNear(v, 5, 1e-9) {
		t.Errorf("Hausdorff = %g", v)
	}
	if v := (CentroidDistance{Cfg: cfg}).Between(a, b); !metricsTestNear(v, 13.0/3.0, 5e-4) {
		t.Errorf("Centroid = %g", v)
	}
	// Носители не пересекаются: интеграл равен сумме площадей 1 + 2 = 3.
	if v := (HammingDistance{Cfg: cfg}).Between(a, b); !metricsTestNear(v, 3, 0.02) {
		t.Errorf("Hamming = %g", v)
	}
}

func TestDistanceProperties(t *testing.T) {
	a, b := metricsTestU(t, 1, 4), metricsTestU(t, 2, 9)
	for _, d := range []Distance{
		CutsL1Distance{}, CutsL2Distance{}, HausdorffDistance{}, CentroidDistance{}, HammingDistance{},
	} {
		if v := d.Between(a, a); !metricsTestNear(v, 0, 1e-9) {
			t.Errorf("%s: d(a,a) = %g", d.Name(), v)
		}
		if d1, d2 := d.Between(a, b), d.Between(b, a); !metricsTestNear(d1, d2, 1e-9) || d1 <= 0 {
			t.Errorf("%s: d(a,b)=%g d(b,a)=%g", d.Name(), d1, d2)
		}
	}
}

func TestDistanceMatrix(t *testing.T) {
	items := []Number{metricsTestU(t, 0, 1), metricsTestU(t, 2, 3), metricsTestU(t, 10, 12)}
	m := DistanceMatrix(items, CutsL1Distance{})
	for i := range m {
		if m[i][i] != 0 {
			t.Errorf("diag[%d] = %g", i, m[i][i])
		}
		for j := range m {
			if m[i][j] != m[j][i] {
				t.Errorf("not symmetric at %d,%d", i, j)
			}
		}
	}
	if !(m[0][1] < m[0][2]) {
		t.Error("closer pair must have smaller distance")
	}
}
