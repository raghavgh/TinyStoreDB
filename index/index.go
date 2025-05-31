package index

import "github.com/raghavgh/TinyStoreDB/pair"

type Index[K comparable, V any] interface {
	// Put adds a key-value pair to the index.
	Put(key K, value V)

	// Get retrieves the value associated with the key.
	Get(key K) (V, bool)

	// Delete removes the key-value pair from the index.
	Delete(key K)

	// Rebuild rebuilds the index.
	Rebuild(pairList pair.List[K, V])

	// GetCurrentSnapshot returns the current snapshot of the index.
	GetCurrentSnapshot() pair.List[K, V]

	// Clear clears the index.
	Clear()

	// Lock locks the index for writing.
	Lock()

	// Unlock unlocks the index.
	Unlock()
}
