package experiment

import (
	"math"
	"reflect"
	"sync"
	"testing"

	"fuzzylab/internal/clustering"
	"fuzzylab/internal/fuzzy"
)

type memoryJournal struct{ generations, clusters, operations int }

func (j *memoryJournal) Log(_ string, _ any) error { j.generations++; return nil }
func (j *memoryJournal) Clustering(_ string, _ map[string]any, _ clustering.Result) error {
	j.clusters++
	return nil
}
func (j *memoryJournal) Operation(_ string, _, _, _ fuzzy.Number, _ error) error {
	j.operations++
	return nil
}

func smallExperiment(t *testing.T) (*Service, *memoryJournal) {
	t.Helper()
	j := &memoryJournal{}
	s, err := New(t.TempDir(), j)
	if err != nil {
		t.Fatal(err)
	}
	f := testGeneration()
	f.N = 12
	if err = s.Generate(f); err != nil {
		t.Fatal(err)
	}
	return s, j
}

func TestGenerationIsReproducibleAndSnapshotsAreIndependent(t *testing.T) {
	s, _ := smallExperiment(t)
	first := s.Snapshot()
	settings := *first.Generation
	first.Items[0] = nil
	first.Generation.Seed = -100
	if s.Snapshot().Items[0] == nil || s.Snapshot().Generation.Seed == -100 {
		t.Fatal("snapshot leaked mutable service state")
	}
	before := s.Snapshot()
	if err := s.Generate(settings); err != nil {
		t.Fatal(err)
	}
	after := s.Snapshot()
	if !reflect.DeepEqual(before.Items, after.Items) || after.Revision != before.Revision+1 {
		t.Fatal("same seed did not reproduce values or advance revision")
	}
	settings.Seed++
	if err := s.Generate(settings); err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(after.Items, s.Snapshot().Items) {
		t.Fatal("different seed did not change dataset")
	}
}

func TestClusteringCacheUsesControlsAndDatasetRevision(t *testing.T) {
	s, j := smallExperiment(t)
	c := DefaultControls()
	c.K = 3
	first, err := s.Cluster(c)
	if err != nil {
		t.Fatal(err)
	}
	first.Clusters.Labels[0] = -1
	first.Clusters.Medoids[0] = -1
	first.Dataset.Items[0] = nil
	second, err := s.Cluster(c)
	if err != nil {
		t.Fatal(err)
	}
	if j.clusters != 1 || second.Clusters.Labels[0] < 0 || second.Clusters.Medoids[0] < 0 || second.Dataset.Items[0] == nil {
		t.Fatal("cache is not reused safely")
	}
	c.Alpha = 0.5
	if _, err = s.Cluster(c); err != nil {
		t.Fatal(err)
	}
	if j.clusters != 2 {
		t.Fatal("alpha did not invalidate calculation")
	}
	f := *s.Snapshot().Generation
	f.Seed++
	if err = s.Generate(f); err != nil {
		t.Fatal(err)
	}
	third, err := s.Cluster(c)
	if err != nil {
		t.Fatal(err)
	}
	if j.clusters != 3 || third.Dataset.Revision == second.Dataset.Revision {
		t.Fatal("dataset replacement reused stale clustering")
	}
}

func TestInvalidControlsDoNotProduceResults(t *testing.T) {
	s, _ := smallExperiment(t)
	for _, c := range []Controls{{Algo: "pam", K: 2, Alpha: math.NaN()}, {Algo: "pam", K: 2, Alpha: 1.1}, {Algo: "pam", K: 0}, {Algo: "other", K: 2}, {Algo: "agglo", Threshold: 0}, {Algo: "agglo", Threshold: math.Inf(1)}} {
		if _, err := s.Cluster(c); err == nil {
			t.Errorf("accepted %+v", c)
		}
	}
}

func TestArithmeticCacheReturnsIndependentResults(t *testing.T) {
	s, j := smallExperiment(t)
	f := OperationRequest{Mu1: 0, Sigma1: 0.3, Mu2: 0.7, Sigma2: 0.4, Op: "add"}
	first, err := s.Calculate(f)
	if err != nil {
		t.Fatal(err)
	}
	first.Names[0] = "changed"
	first.Items[0] = nil
	first.Characteristics[0].Mean = -1
	second, err := s.Calculate(f)
	if err != nil {
		t.Fatal(err)
	}
	if j.operations != 1 || second.Items[0] == nil || second.Names[0] != "A" || second.Characteristics[0].Mean < 0 {
		t.Fatal("arithmetic cache leaked state")
	}
	f.Op = "div"
	if _, err = s.Calculate(f); err != nil {
		t.Fatal(err)
	}
	f.Op = "unknown"
	if _, err = s.Calculate(f); err == nil {
		t.Fatal("accepted unknown operation")
	}
}

func TestServiceConcurrentSnapshotsAndCalculations(t *testing.T) {
	s, _ := smallExperiment(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 4; j++ {
				snapshot := s.Snapshot()
				snapshot.Items[0] = nil
				if _, err := s.Cluster(DefaultControls()); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if s.Snapshot().Items[0] == nil {
		t.Fatal("concurrent snapshot mutation affected dataset")
	}
}
