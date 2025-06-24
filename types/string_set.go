package types

// StringSet represents a set of unique strings
type StringSet struct {
	data map[string]struct{}
}

// NewStringSet initializes a new StringSet
func NewStringSet() *StringSet {
	return &StringSet{
		data: make(map[string]struct{}),
	}
}

// Add inserts a new key into the set
func (s *StringSet) Add(key string) {
	s.data[key] = struct{}{}
}

// Remove deletes a key from the set
func (s *StringSet) Remove(key string) {
	delete(s.data, key)
}

// Has checks if a key exists in the set
func (s *StringSet) Has(key string) bool {
	_, exists := s.data[key]
	return exists
}

// Keys returns all keys in the set as a slice
func (s *StringSet) Keys() []string {
	keys := make([]string, 0, len(s.data))
	for key := range s.data {
		keys = append(keys, key)
	}
	return keys
}

// Size returns the number of elements in the set
func (s *StringSet) Size() int {
	return len(s.data)
}

// Clear removes all elements from the set
func (s *StringSet) Clear() {
	s.data = make(map[string]struct{})
}
