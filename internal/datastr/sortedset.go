package datastr

// Currently not goroutine safe
type SortedSet struct {
	hashmap  map[string]*node
	skiplist *SkipList
}

func NewSortedSet() *SortedSet {
	return &SortedSet{
		hashmap:  make(map[string]*node),
		skiplist: NewSkipList(),
	}
}

func (s *SortedSet) Add(key string) {
	n := s.skiplist.Insert([]byte(key), nil)
	s.hashmap[key] = n
}

func (s *SortedSet) Remove(key string) bool {
	_, err := s.skiplist.Delete([]byte(key))
	if err != nil {
		return false
	}
	delete(s.hashmap, key)
	return true
}

func (s *SortedSet) Contains(key string) (bool, *node) {
	if node, found := s.hashmap[key]; found {
		return true, node
	}
	return false, nil
}

func (s *SortedSet) Len() int { return len(s.hashmap) }

func (s *SortedSet) Iterator() Iterator { return &skipIterator{curr: s.skiplist.head} }

func (s *SortedSet) ForEachKey(fn func(key string) error) {
	for n := s.skiplist.head.tower[0]; n != nil; n = n.tower[0] {
		fn(string(n.key))
	}
}
