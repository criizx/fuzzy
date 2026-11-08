package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"fuzzylab/fuzzy/core"
	"fuzzylab/gen"
)

func TestRoundTrip(t *testing.T) {
	items, err := gen.Random(gen.Params{N: 100, Mu: gen.Range{Lo: -1, Hi: 2}, Sigma: gen.Range{Lo: 0.15, Hi: 0.6}, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "data", "set.csv")
	if err := Save(path, items); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(items) {
		t.Fatalf("len = %d, want %d", len(got), len(items))
	}
	for i := range items {
		if got[i].Support() != items[i].Support() {
			t.Fatalf("item %d: %v, want %v", i, got[i].Support(), items[i].Support())
		}
	}
}

func TestLoadInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.csv")
	for _, body := range []string{
		"kind,mu,sigma\nlognormal,x,0.4\n",
		"kind,mu,sigma\nlognormal,3,0\n",
		"kind,mu,sigma\ntriangle,1,2\n",
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); !errors.Is(err, core.ErrInvalidParams) {
			t.Errorf("%q: err = %v", body, err)
		}
	}
}
