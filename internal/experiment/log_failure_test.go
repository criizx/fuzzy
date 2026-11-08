package experiment

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"fuzzylab/internal/generator"
	"fuzzylab/internal/logger"
	"fuzzylab/internal/storage"
)

func logFailureService(t *testing.T) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "log.json")
	l, err := logger.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(dir, l)
	if err != nil {
		l.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { s.log.(*logger.Logger).Close() })
	return s, path
}

func reopenLog(t *testing.T, s *Service, path string) {
	t.Helper()
	l, err := logger.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s.log = l
}

func testGeneration() Generation {
	f := DefaultGeneration()
	f.Mode = "random"
	f.N = 120
	f.MuLo = 0
	f.MuHi = 2
	f.SigmaLo = 0.2
	f.SigmaHi = 0.5
	f.Seed = 7
	return f
}

func TestGenerationLogFailurePreservesStateUntilRetry(t *testing.T) {
	s, path := logFailureService(t)
	previous := s.Snapshot()
	s.log.(*logger.Logger).Close()
	for attempt := 0; attempt < 2; attempt++ {
		if err := s.Generate(testGeneration()); err == nil {
			t.Fatal("closed journal accepted generation")
		}
		if !reflect.DeepEqual(s.Snapshot(), previous) {
			t.Fatal("failed generation changed dataset or metadata")
		}
	}
	reopenLog(t, s, path)
	if err := s.Generate(testGeneration()); err != nil {
		t.Fatal(err)
	}
	got := s.Snapshot()
	if len(got.Items) != 120 || *got.Generation != testGeneration() || got.Revision != previous.Revision+1 {
		t.Fatal("retry failed to commit full dataset")
	}
}

func TestLoadLogFailurePreservesSetUntilRetry(t *testing.T) {
	s, path := logFailureService(t)
	items, err := generator.Random(generator.Params{N: 120, Mu: generator.Range{Lo: 0, Hi: 2}, Sigma: generator.Range{Lo: 0.2, Hi: 0.5}, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	if err = storage.Save(filepath.Join(s.dir, "input", "review.csv"), items); err != nil {
		t.Fatal(err)
	}
	previous := s.Snapshot()
	s.log.(*logger.Logger).Close()
	for attempt := 0; attempt < 2; attempt++ {
		if _, err = s.Load("review.csv"); err == nil {
			t.Fatal("closed journal accepted import")
		}
		if !reflect.DeepEqual(s.Snapshot(), previous) {
			t.Fatal("failed import changed dataset")
		}
	}
	reopenLog(t, s, path)
	if _, err = s.Load("review.csv"); err != nil {
		t.Fatal(err)
	}
	got := s.Snapshot()
	if !reflect.DeepEqual(got.Items, items) || got.Generation != nil || got.Source != "review.csv" {
		t.Fatal("import invented generation metadata or lost values")
	}
}

func TestFailedJournalWriteIsRetriedForCalculations(t *testing.T) {
	for _, op := range []bool{false, true} {
		t.Run(map[bool]string{false: "clustering", true: "arithmetic"}[op], func(t *testing.T) {
			s, path := logFailureService(t)
			invoke := func() error {
				if op {
					_, err := s.Calculate(OperationRequest{Mu1: 0, Sigma1: 0.3, Mu2: 0.7, Sigma2: 0.4, Op: "add"})
					return err
				}
				_, err := s.Cluster(DefaultControls())
				return err
			}
			s.log.(*logger.Logger).Close()
			for attempt := 0; attempt < 2; attempt++ {
				if err := invoke(); err == nil {
					t.Fatal("failed audit was incorrectly cached")
				}
			}
			reopenLog(t, s, path)
			for attempt := 0; attempt < 2; attempt++ {
				if err := invoke(); err != nil {
					t.Fatal(err)
				}
			}
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			dec := json.NewDecoder(bytes.NewReader(body))
			count := 0
			for dec.More() {
				var row map[string]any
				if err = dec.Decode(&row); err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count != 2 {
				t.Fatalf("got %d entries, want startup and one calculation", count)
			}
		})
	}
}
