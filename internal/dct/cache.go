package dct

import (
	"fmt"
	"sync"
)

type Cache struct {
	data sync.Map
}

func NewCache() *Cache {
	var c Cache
	return &c
}

func (c *Cache) New(w, h int) *DCT {
	key := fmt.Sprintf("%d-%d", w, h)
	if v, ok := c.data.Load(key); ok {
		return v.(*DCT)
	}
	dct := New(w, h)
	actual, loaded := c.data.LoadOrStore(key, dct)
	if loaded {
		return actual.(*DCT)
	}
	return dct
}
