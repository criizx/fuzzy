package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"fuzzylab/gen"
	"fuzzylab/logger"
	"fuzzylab/storage"
)

func logFailureServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "log.json")
	l, err := logger.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(dir, l)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.log.Close() })
	return s, path
}

func reopenReviewLog(t *testing.T, s *Server, path string) {
	t.Helper()
	l, err := logger.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s.log = l
}

func generationRequest(mode string) *http.Request {
	r := httptest.NewRequest("POST", "/generate", nil)
	r.Form = url.Values{"mode": {mode}, "n": {"120"}, "mulo": {"0"}, "muhi": {"2"}, "sigmalo": {"0.2"}, "sigmahi": {"0.5"}, "seed": {"7"}}
	return r
}

func TestGenerationLogFailurePreservesStateUntilRetry(t *testing.T) {
	s, path := logFailureServer(t)
	previous := s.items
	previousForm := s.genForm
	s.lastLogged = "previous successful clustering"
	if err := s.log.Close(); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := s.onGenerate(generationRequest("random")); err == nil {
			t.Fatal("closed journal must reject every attempt")
		}
		if !reflect.DeepEqual(s.items, previous) || s.genForm != previousForm || s.lastLogged != "previous successful clustering" {
			t.Fatal("failed generation changed experiment state")
		}
	}
	reopenReviewLog(t, s, path)
	if _, err := s.onGenerate(generationRequest("random")); err != nil {
		t.Fatal(err)
	}
	if len(s.items) != 120 || s.genForm.N != 120 || s.genForm.Mode != "random" || s.lastLogged != "" {
		t.Fatal("successful retry did not commit complete experiment state")
	}
}

func TestLoadLogFailurePreservesSetUntilRetry(t *testing.T) {
	s, path := logFailureServer(t)
	items, err := gen.Random(gen.Params{N: 120, Mu: gen.Range{Lo: 0, Hi: 2}, Sigma: gen.Range{Lo: 0.2, Hi: 0.5}, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Save(filepath.Join(s.dir, "input", "review.csv"), items); err != nil {
		t.Fatal(err)
	}
	previous := s.items
	previousForm := s.genForm
	s.lastLogged = "previous successful clustering"
	s.log.Close()
	r := httptest.NewRequest("POST", "/load", nil)
	r.Form = url.Values{"file": {"review.csv"}}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := s.onLoad(r); err == nil {
			t.Fatal("closed journal must reject load")
		}
		if !reflect.DeepEqual(s.items, previous) || s.genForm != previousForm || s.lastLogged != "previous successful clustering" {
			t.Fatal("failed load changed experiment state")
		}
	}
	reopenReviewLog(t, s, path)
	if _, err := s.onLoad(r); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.items, items) || s.lastLogged != "" {
		t.Fatal("successful load retry did not commit set")
	}
}

func TestFailedJournalWriteIsRetriedForCalculations(t *testing.T) {
	for _, operation := range []bool{false, true} {
		t.Run(map[bool]string{false: "clustering", true: "arithmetic"}[operation], func(t *testing.T) {
			s, path := logFailureServer(t)
			invoke := func() error {
				if operation {
					_, err := s.calcOps(defaultOps)
					return err
				}
				_, err := s.cluster(defaultCtl)
				return err
			}
			s.log.Close()
			for attempt := 0; attempt < 2; attempt++ {
				if err := invoke(); err == nil {
					t.Fatal("retry falsely succeeded without writable journal")
				}
			}
			if s.lastLogged != "" || s.lastOp != "" {
				t.Fatal("failed journal write was marked as logged")
			}
			reopenReviewLog(t, s, path)
			if err := invoke(); err != nil {
				t.Fatal(err)
			}
			if err := invoke(); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			// The startup generation and exactly one successful calculation are logged.
			dec := json.NewDecoder(bytes.NewReader(body))
			count := 0
			for dec.More() {
				var row map[string]any
				if err := dec.Decode(&row); err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count != 2 {
				t.Fatalf("logged %d entries, want startup plus one calculation", count)
			}
		})
	}
}

func TestRandomGenerationIgnoresCloudOnlyFields(t *testing.T) {
	s, _ := logFailureServer(t)
	r := generationRequest("random")
	r.Form.Set("groups", "invalid")
	r.Form.Set("noise", "invalid")
	if _, err := s.onGenerate(r); err != nil {
		t.Fatal(err)
	}
	if s.genForm.Groups != defaultGen.Groups || s.genForm.Noise != defaultGen.Noise {
		t.Fatal("random mode must retain usable cloud defaults")
	}
	r = generationRequest("clouds")
	if _, err := s.onGenerate(r); err == nil {
		t.Fatal("clouds require groups and noise")
	}
	r = generationRequest("random")
	r.Form.Set("mulo", "invalid")
	if _, err := s.onGenerate(r); err == nil {
		t.Fatal("random mode must validate common fields")
	}
}
