package fuzzy

import (
	"errors"
	"testing"
)

func setTestU(t *testing.T, a, b float64) Number {
	t.Helper()
	x, err := NewUniform(a, b)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func TestCollection(t *testing.T) {
	c := NewCollection(setTestU(t, 0, 1), setTestU(t, 5, 8))
	c.Add(setTestU(t, -2, 0.5))
	if c.Len() != 3 {
		t.Fatalf("Len = %d", c.Len())
	}

	b, err := c.Bounds()
	if err != nil || b != (Interval{Lo: -2, Hi: 8}) {
		t.Errorf("Bounds = %v, %v", b, err)
	}

	if _, err := c.Get(3); !errors.Is(err, ErrIndex) {
		t.Errorf("Get(3) err = %v", err)
	}
	if err := c.Remove(1); err != nil || c.Len() != 2 {
		t.Errorf("Remove: %v, len %d", err, c.Len())
	}

	wide := c.Filter(func(n Number) bool { return n.Support().Width() > 1 })
	if wide.Len() != 1 {
		t.Errorf("Filter len = %d", wide.Len())
	}

	// Slice возвращает копию
	s := c.Slice()
	s[0] = nil
	if g, _ := c.Get(0); g == nil {
		t.Error("Slice must return a copy")
	}

	if _, err := NewCollection().Bounds(); !errors.Is(err, ErrEmpty) {
		t.Errorf("empty Bounds err = %v", err)
	}
}
