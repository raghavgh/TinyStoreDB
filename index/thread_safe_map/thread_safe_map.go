package thread_safe_map

import (
	"sync"

	"github.com/raghavgh/TinyStoreDB/pair"
)

func (gsm *GoThreadSafeMap[K, V]) GetCurrentSnapshot() pair.List[K, V] {
	gsm.RLock()
	defer gsm.RUnlock()
	pairList := make(pair.List[K, V], 0, len(gsm.m))
	for k, v := range gsm.m {
		pairList = append(pairList, &pair.Pair[K, V]{Key: k, Value: v})
	}
	return pairList
}

func (gsm *GoThreadSafeMap[K, V]) Rebuild(pairList pair.List[K, V]) {
	gsm.Lock()
	defer gsm.Unlock()
	gsm.m = make(map[K]V)
	for _, p := range pairList {
		gsm.m[p.Key] = p.Value
	}
}

func (gsm *GoThreadSafeMap[K, V]) Delete(key K) {
	gsm.Lock()
	defer gsm.Unlock()
	delete(gsm.m, key)
}

func (gsm *GoThreadSafeMap[K, V]) Clear() {
	gsm.Lock()
	defer gsm.Unlock()
	for k := range gsm.m {
		delete(gsm.m, k)
	}
}

func (gsm *GoThreadSafeMap[K, V]) Get(key K) (V, bool) {
	gsm.RLock()
	defer gsm.RUnlock()
	value, ok := gsm.m[key]
	return value, ok
}

func (gsm *GoThreadSafeMap[K, V]) Put(key K, value V) {
	gsm.Lock()
	defer gsm.Unlock()
	gsm.m[key] = value
}

func New[K comparable, V any]() *GoThreadSafeMap[K, V] {
	return &GoThreadSafeMap[K, V]{
		m: make(map[K]V),
	}
}

type GoThreadSafeMap[K comparable, V any] struct {
	m map[K]V
	sync.RWMutex
}
