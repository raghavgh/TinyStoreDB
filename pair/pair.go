package pair

// NewPair creates a new pair
func NewPair[K comparable, V any](key K, value V) *Pair[K, V] {
	return &Pair[K, V]{Key: key, Value: value}
}

// Pair represents a key-value pair
type Pair[K comparable, V any] struct {
	Key   K
	Value V
}

// List PairList is a list of pairs
type List[K comparable, V any] []*Pair[K, V]
