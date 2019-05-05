package count

import (
	"sync"

	"github.com/ngalaiko/words/common"
)

type Stream struct {
	n int

	frequencyMap map[string]uint64
	guard        *sync.RWMutex
}

type Element struct {
	Key   string
	Count uint64
}

func New(n int) *Stream {
	return &Stream{
		n:            n,
		frequencyMap: make(map[string]uint64, len(common.Map)),
		guard:        &sync.RWMutex{},
	}
}

func (c *Stream) Keys() []Element {
	c.guard.RLock()
	defer c.guard.RUnlock()

	// NOTE:
	// 2^25 = 33554432
	// assume it's larger then a number of occurrences for the most frequent word
	freq := make([]*string, 2<<25)
	for word, count := range c.frequencyMap {
		wCopy := word
		freq[count] = &wCopy
	}

	res := make([]Element, 0, 10)
	for i := uint64(len(freq) - 1); i >= 0 && len(res) < c.n; i-- {
		if freq[i] == nil {
			continue
		}
		res = append(res, Element{
			Key:   *freq[i],
			Count: i,
		})
	}

	return res
}

func (c *Stream) Insert(word string) {
	if !common.Map[word] {
		return
	}
	c.guard.Lock()
	c.frequencyMap[word]++
	c.guard.Unlock()
}
