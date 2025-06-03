package datastr

import (
	"iter"
	"slices"
)

type Set struct {
	arr []string
}

func NewSet() *Set {
	return &Set{}
}

// Add value in the slice if not found in slice
func (s *Set) Add(val string) {
	found := slices.Contains(s.arr, val)
	if found {
		return
	}
	s.arr = append(s.arr, val)
}

// Remove value if found in the slice
func (s *Set) Remove(val string) {
	found := slices.Contains(s.arr, val)
	if !found {
		return
	}
	idx := slices.Index(s.arr, val)
	s.arr = slices.Delete(s.arr, idx, idx)
}

// Return the iterator on the Set
func (s *Set) All() iter.Seq2[int, string] {
	return slices.All(s.arr)
}

func (s *Set) GetAll() []string {
	return s.arr
}

func (s *Set) IsEmpty() bool {
	return len(s.arr) == 0
}
