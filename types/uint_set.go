package types

// UintSet represents a set of unique uint values
type UintSet struct {
	data map[uint]struct{}
}

// NewUintSet initializes a new UintSet
func NewUintSet() *UintSet {
	return &UintSet{
		data: make(map[uint]struct{}),
	}
}

// Add inserts a new value into the set
func (s *UintSet) Add(value uint) {
	s.data[value] = struct{}{}
}

// Remove deletes a value from the set
func (s *UintSet) Remove(value uint) {
	delete(s.data, value)
}

// Has checks if a value exists in the set
func (s *UintSet) Has(value uint) bool {
	_, exists := s.data[value]
	return exists
}

// Values returns all values in the set as a slice
func (s *UintSet) Values() []uint {
	values := make([]uint, 0, len(s.data))
	for value := range s.data {
		values = append(values, value)
	}
	return values
}

// Size returns the number of elements in the set
func (s *UintSet) Size() int {
	return len(s.data)
}

// Clear removes all elements from the set
func (s *UintSet) Clear() {
	s.data = make(map[uint]struct{})
}
