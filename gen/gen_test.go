package gen

import (
	"errors"
	"testing"

	"fuzzylab/fuzzy/core"
	"fuzzylab/fuzzy/number"
)

var params = Params{N: 150, Mu: Range{-1, 2}, Sigma: Range{0.15, 0.6}, Seed: 1}

func TestUniform(t *testing.T) {
	items, err := Random(params)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != params.N {
		t.Fatalf("len = %d, want %d", len(items), params.N)
	}
	for _, it := range items {
		u := it.(number.LogNormal)
		if u.Mu < params.Mu.Lo || u.Mu > params.Mu.Hi || u.Sigma < params.Sigma.Lo || u.Sigma > params.Sigma.Hi {
			t.Fatalf("out of ranges: %v", u)
		}
	}
	again, _ := Random(params)
	if items[7].Support() != again[7].Support() {
		t.Error("same seed must give same data")
	}
}

func TestClouds(t *testing.T) {
	items, labels, err := Clouds(CloudParams{Params: params, K: 4, Noise: 0.02})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != params.N || len(labels) != params.N {
		t.Fatalf("len = %d/%d, want %d", len(items), len(labels), params.N)
	}
	seen := map[int]bool{}
	for _, l := range labels {
		seen[l] = true
	}
	if len(seen) != 4 {
		t.Errorf("clusters = %d, want 4", len(seen))
	}
}

func TestInvalid(t *testing.T) {
	bad := []Params{
		{N: 0, Mu: Range{0, 1}, Sigma: Range{1, 2}},
		{N: 10, Mu: Range{1, 0}, Sigma: Range{1, 2}},
		{N: 10, Mu: Range{0, 1}, Sigma: Range{0, 2}},
	}
	for _, p := range bad {
		if _, err := Random(p); !errors.Is(err, core.ErrInvalidParams) {
			t.Errorf("%+v: err = %v", p, err)
		}
	}
	if _, _, err := Clouds(CloudParams{Params: params, K: 0}); !errors.Is(err, core.ErrInvalidParams) {
		t.Errorf("K=0: err = %v", err)
	}
}
