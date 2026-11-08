package fuzzy

import (
	"fmt"
)

// Collection stores an ordered sequence of fuzzy values.
type Collection struct {
	items []Number
}

func NewCollection(items ...Number) *Collection {
	return &Collection{items: append([]Number(nil), items...)}
}

func (c *Collection) Add(items ...Number) { c.items = append(c.items, items...) }
func (c *Collection) Len() int            { return len(c.items) }

func (c *Collection) Get(i int) (Number, error) {
	if i < 0 || i >= len(c.items) {
		return nil, fmt.Errorf("%w: %d (размер %d)", ErrIndex, i, len(c.items))
	}
	return c.items[i], nil
}

func (c *Collection) Remove(i int) error {
	if i < 0 || i >= len(c.items) {
		return fmt.Errorf("%w: %d (размер %d)", ErrIndex, i, len(c.items))
	}
	c.items = append(c.items[:i], c.items[i+1:]...)
	return nil
}

// Slice returns a copy of the collection contents.
func (c *Collection) Slice() []Number { return append([]Number(nil), c.items...) }

func (c *Collection) Clone() *Collection { return &Collection{items: c.Slice()} }

func (c *Collection) Filter(pred func(Number) bool) *Collection {
	out := &Collection{}
	for _, it := range c.items {
		if pred(it) {
			out.items = append(out.items, it)
		}
	}
	return out
}

// Bounds returns the smallest interval enclosing all supports.
// An empty collection returns ErrEmpty.
func (c *Collection) Bounds() (Interval, error) {
	if len(c.items) == 0 {
		return Interval{}, ErrEmpty
	}
	b := c.items[0].Support()
	for _, it := range c.items[1:] {
		b = b.Hull(it.Support())
	}
	return b, nil
}
