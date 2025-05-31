package go_map

import "github.com/raghavgh/TinyStoreDB/pair"

func (gm *GoMap[K, V]) Clear() {
	for k := range gm.m {
		delete(gm.m, k)
	}
}

func (gm *GoMap[K, V]) Delete(key K) {
	delete(gm.m, key)
}

func (gm *GoMap[K, V]) Put(key K, value V) {
	gm.m[key] = value
}

func (gm *GoMap[K, V]) Get(key K) (V, bool) {
	v, ok := gm.m[key]
	return v, ok
}

func (gm *GoMap[K, V]) Rebuild(pairList pair.List[K, V]) {
	gm.m = make(map[K]V)
	for _, p := range pairList {
		gm.Put(p.Key, p.Value)
	}
}

func New[K comparable, V any]() *GoMap[K, V] {
	return &GoMap[K, V]{
		m: make(map[K]V),
	}
}

type GoMap[K comparable, V any] struct {
	m map[K]V
}
