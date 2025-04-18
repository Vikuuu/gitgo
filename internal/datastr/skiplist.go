package datastr

import (
	"bytes"
	"errors"
	"math"
	_ "unsafe"
)

const (
	MaxHeight = 16
	PValue    = 0.5
)

var (
	ErrKeyNotFound = errors.New("key not found")
	probabilities  [MaxHeight]uint32
)

type Iterator interface {
	Next() bool
	Key() string
}

type node struct {
	key   []byte
	val   []byte
	tower [MaxHeight]*node
}

type SkipList struct {
	head   *node
	height int
}

type skipIterator struct {
	curr *node
}

func NewSkipList() *SkipList {
	return &SkipList{
		head:   &node{},
		height: 1,
	}
}

// search for the key, while search keep the journey array of
// nodes traversed, if found return node pointer else nil
func (s *SkipList) search(key []byte) (*node, [MaxHeight]*node) {
	var next *node
	var journey [MaxHeight]*node
	prev := s.head
	for lvl := MaxHeight - 1; lvl >= 0; lvl-- {
		for next = prev.tower[lvl]; next != nil; next = prev.tower[lvl] {
			if bytes.Compare(key, next.key) <= 0 {
				break
			}
			prev = next
		}
		journey[lvl] = prev
	}

	if next != nil && bytes.Equal(key, next.key) {
		return next, journey
	}

	return nil, journey
}

// If the key found, returns its value else returns error
func (s *SkipList) Find(key []byte) ([]byte, error) {
	found, _ := s.search(key)
	if found == nil {
		return nil, ErrKeyNotFound
	}
	return found.val, nil
}

// First searches the skiplist for the key, if key found
// overwrites its value, if not found then adds that value to
// the skiplist
func (s *SkipList) Insert(key []byte, val []byte) *node {
	found, journey := s.search(key)
	if found != nil {
		found.val = val
		return found
	}
	height := randomHeight()
	n := &node{key: key, val: val}

	for lvl := 0; lvl < height; lvl++ {
		prev := journey[lvl]
		if prev == nil {
			// prev is nil if we are extending the height of the tree
			// because that level did not exist while the journey was
			// being recorded
			prev = s.head
		}
		n.tower[lvl] = prev.tower[lvl]
		prev.tower[lvl] = n
	}

	if height > s.height {
		s.height = height
	}
	return n
}

// Delete the key-value if found, else return err
// shrink the tower size if neccessary
func (s *SkipList) Delete(key []byte) (bool, error) {
	found, journey := s.search(key)
	if found == nil {
		return false, ErrKeyNotFound
	}

	for lvl := 0; lvl < s.height; lvl++ {
		if journey[lvl].tower[lvl] != found {
			break
		}
		journey[lvl].tower[lvl] = found.tower[lvl]
		found.tower[lvl] = nil
	}
	found = nil
	s.shrink()
	return true, nil
}

func (s *SkipList) shrink() {
	for lvl := s.height - 1; lvl >= 0; lvl-- {
		if s.head.tower[lvl] == nil {
			s.height--
		}
	}
}

func (it *skipIterator) Next() bool {
	if it.curr == nil {
		return false
	}
	it.curr = it.curr.tower[0]
	return it.curr != nil
}

func (it *skipIterator) Key() string {
	return string(it.curr.key)
}

// The below function is needed to generate a random number
// this is being used because this is more efficient the
// math.rand
//
//go:linkname Uint32 runtime.fastrand
func Uint32() uint32

func init() {
	probability := 1.0
	for lvl := 0; lvl < MaxHeight; lvl++ {
		probabilities[lvl] = uint32(probability * float64(math.MaxUint32))
		probability *= PValue
	}
}

func randomHeight() int {
	seed := Uint32()
	height := 1
	for height < MaxHeight && seed <= probabilities[height] {
		height++
	}
	return height
}
