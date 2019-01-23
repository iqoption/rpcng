package consistenthash

import (
	"sort"
	"strconv"

	"github.com/segmentio/fasthash/fnv1a"
)

type Map struct {
	replicas int
	keys     []uint64 // Sorted
	hashMap  map[uint64]string
}

func New(replicas int) *Map {
	m := &Map{
		replicas: replicas,
		hashMap:  make(map[uint64]string),
	}
	return m
}

// Returns true if there are no items available.
func (m *Map) IsEmpty() bool {
	return len(m.keys) == 0
}

// Adds some keys to the hash.
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := fnv1a.HashString64(strconv.Itoa(i) + key)
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Slice(m.keys, func(i, j int) bool { return m.keys[i] < m.keys[j] })
}

// Gets the closest item in the hash to the provided key.
func (m *Map) Get(key string) string {
	if m.IsEmpty() {
		return ""
	}

	hash := fnv1a.HashString64(key)

	// Binary search for appropriate replica.
	idx := sort.Search(len(m.keys), func(i int) bool { return m.keys[i] >= hash })
	// Means we have cycled back to the first replica.
	if idx == len(m.keys) {
		idx = 0
	}

	return m.hashMap[m.keys[idx]]
}

func (m *Map) GetUint64(key uint64) string {
	if m.IsEmpty() {
		return ""
	}

	hash := fnv1a.HashUint64(key)

	// Binary search for appropriate replica.
	idx := sort.Search(len(m.keys), func(i int) bool { return m.keys[i] >= hash })

	// Means we have cycled back to the first replica.
	if idx == len(m.keys) {
		idx = 0
	}

	return m.hashMap[m.keys[idx]]
}
