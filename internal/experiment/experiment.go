// Package experiment coordinates reproducible datasets, calculations, persistence, and audit logging.
// It has no dependency on HTTP, templates, or chart geometry.
package experiment

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"sync"

	"fuzzylab/internal/clustering"
	"fuzzylab/internal/fuzzy"
	"fuzzylab/internal/generator"
	"fuzzylab/internal/storage"
)

// Journal is the audit boundary. Failed writes must not commit a new dataset.
type Journal interface {
	Log(action string, data any) error
	Clustering(algo string, params map[string]any, result clustering.Result) error
	Operation(op string, a, b, result fuzzy.Number, opErr error) error
}

// Generation defines the parameter ranges and seed used to create a dataset.
// Uniform sampling of Mu and Sigma does not change the lognormal model of X.
type Generation struct {
	Mode             string
	N, Groups        int
	MuLo, MuHi       float64
	SigmaLo, SigmaHi float64
	Noise            float64
	Seed             int64
}

// Controls selects the alpha-cut distance and clustering algorithm.
type Controls struct {
	Alpha     float64
	Algo      string
	K         int
	Threshold float64
}

// DefaultGeneration returns an independent copy of the initial settings.
func DefaultGeneration() Generation {
	return Generation{Mode: "clouds", N: 200, Groups: 4, MuLo: -1, MuHi: 2, SigmaLo: 0.15, SigmaHi: 0.6, Noise: 0.03, Seed: 1}
}

// DefaultControls returns the initial clustering settings.
func DefaultControls() Controls { return Controls{Algo: "pam", K: 4, Threshold: 1} }

// Dataset is a snapshot. Generation is nil for imported CSV data without seed metadata.
// Items contains immutable built-in fuzzy values and is copied at service boundaries.
type Dataset struct {
	Items      []fuzzy.Number
	Generation *Generation
	Revision   uint64
	Source     string
}

// Result associates a dataset snapshot with a clustering run.
type Result struct {
	Dataset  Dataset
	Controls Controls
	Clusters clustering.Result
}

// Service owns the current dataset and a bounded cache of the latest calculation.
// Its methods are safe for concurrent callers.
type Service struct {
	mu       sync.Mutex
	dir      string
	log      Journal
	data     Dataset
	cached   *Result
	lastOp   OperationRequest
	opCached *ArithmeticResult
	opErr    error
}

// New creates a service and commits the default dataset after a successful audit write.
func New(dir string, journal Journal) (*Service, error) {
	if journal == nil {
		return nil, fmt.Errorf("журнал не задан")
	}
	s := &Service{dir: dir, log: journal}
	if err := s.Generate(DefaultGeneration()); err != nil {
		return nil, err
	}
	return s, nil
}

// Generate creates a complete dataset before replacing the current one.
func (s *Service) Generate(f Generation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.Mode != "random" && f.Mode != "clouds" {
		return fmt.Errorf("неизвестный режим генерации")
	}
	p := generator.Params{N: f.N, Mu: generator.Range{Lo: f.MuLo, Hi: f.MuHi}, Sigma: generator.Range{Lo: f.SigmaLo, Hi: f.SigmaHi}, Seed: f.Seed}
	var items []fuzzy.Number
	var err error
	if f.Mode == "random" {
		items, err = generator.Random(p)
	} else {
		items, _, err = generator.Clouds(generator.CloudParams{Params: p, K: f.Groups, Noise: f.Noise})
	}
	if err != nil {
		return err
	}
	if err := s.log.Log("генерация", f); err != nil {
		return err
	}
	s.data = Dataset{Items: items, Generation: &f, Revision: s.data.Revision + 1, Source: "generated"}
	s.cached = nil
	return nil
}

func cloneDataset(d Dataset) Dataset {
	d.Items = append([]fuzzy.Number(nil), d.Items...)
	if d.Generation != nil {
		g := *d.Generation
		d.Generation = &g
	}
	return d
}

func cloneResult(r Result) Result {
	r.Dataset = cloneDataset(r.Dataset)
	r.Clusters.Labels = append([]int(nil), r.Clusters.Labels...)
	r.Clusters.Medoids = append([]int(nil), r.Clusters.Medoids...)
	return r
}

// Snapshot returns an independent snapshot without running any calculation.
func (s *Service) Snapshot() Dataset {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneDataset(s.data)
}

// Cluster calculates and journals a clustering result, or reuses the latest matching run.
// Failed calculations and audit writes are never cached as successful runs.
func (s *Service) Cluster(c Controls) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.data.Items) == 0 {
		return Result{}, fuzzy.ErrEmpty
	}
	if math.IsNaN(c.Alpha) || math.IsInf(c.Alpha, 0) || c.Alpha < 0 || c.Alpha > 1 {
		return Result{}, fmt.Errorf("α должна быть в [0,1]")
	}
	if c.Algo != "pam" && c.Algo != "agglo" {
		return Result{}, fmt.Errorf("неизвестный алгоритм %q", c.Algo)
	}
	if c.Algo == "agglo" && (!(c.Threshold > 0) || math.IsInf(c.Threshold, 0)) {
		return Result{}, fmt.Errorf("порог должен быть положительным и конечным")
	}
	if c.Algo == "pam" {
		if c.K < 1 {
			return Result{}, fmt.Errorf("число кластеров должно быть положительным")
		}
		c.K = min(c.K, len(s.data.Items))
	}
	if s.cached != nil && s.cached.Controls == c {
		return cloneResult(*s.cached), nil
	}
	dm := fuzzy.DistanceMatrix(s.data.Items, fuzzy.AlphaCutDistance{Alpha: c.Alpha})
	var clusters clustering.Result
	var err error
	params := map[string]any{"α": c.Alpha}
	algo := "k-медоидов"
	if c.Algo == "agglo" {
		algo = "иерархический"
		params["порог"] = c.Threshold
		clusters, err = clustering.Agglomerative(dm, c.Threshold)
	} else {
		params["k"] = c.K
		clusters, err = clustering.PAM(dm, c.K)
	}
	if err != nil {
		return Result{}, err
	}
	if err = s.log.Clustering(algo, params, clusters); err != nil {
		return Result{}, err
	}
	r := Result{Dataset: s.data, Controls: c, Clusters: clusters}
	s.cached = &r
	return cloneResult(r), nil
}

// Save writes the current lognormal dataset to a CSV inside the configured data directory.
func (s *Service) Save(name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := dataPath(s.dir, name)
	if err := storage.Save(path, s.data.Items); err != nil {
		return "", err
	}
	return path, s.log.Log("сохранение", map[string]any{"файл": path, "число": len(s.data.Items)})
}

// Load imports a CSV from data/input; it does not invent generation settings or a seed.
func (s *Service) Load(name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := dataPath(filepath.Join(s.dir, "input"), name)
	items, err := storage.Load(path)
	if err != nil {
		return "", err
	}
	if err = s.log.Log("загрузка", map[string]any{"файл": path, "число": len(items)}); err != nil {
		return "", err
	}
	s.data = Dataset{Items: items, Revision: s.data.Revision + 1, Source: filepath.Base(path)}
	s.cached = nil
	return path, nil
}

func dataPath(dir, name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) {
		name = "set"
	}
	if filepath.Ext(name) != ".csv" {
		name += ".csv"
	}
	return filepath.Join(dir, name)
}
