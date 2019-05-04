package count

import (
	"sort"
	"sync"
)

type Stream struct {
	n int

	m      map[string]uint64
	mGuard *sync.RWMutex
}

type Element struct {
	Key   string
	Count uint64
}

func New(n int) *Stream {
	return &Stream{
		n:      n,
		m:      make(map[string]uint64, n),
		mGuard: &sync.RWMutex{},
	}
}

func (c *Stream) Keys() []Element {
	c.mGuard.RLock()
	defer c.mGuard.RUnlock()

	ee := make([]Element, 0, len(c.m))
	for word, count := range c.m {
		ee = append(ee, Element{
			Count: count,
			Key:   word,
		})
	}

	sort.Slice(ee, func(i, j int) bool {
		return ee[i].Count > ee[j].Count
	})

	if len(ee) > c.n {
		return ee[:c.n]
	}
	return ee
}

func (c *Stream) Insert(word string) {
	c.mGuard.Lock()
	c.m[word]++
	c.mGuard.Unlock()
}
