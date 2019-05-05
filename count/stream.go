package count

import (
	"sync/atomic"

	"github.com/cornelk/hashmap"
	"github.com/ngalaiko/words/common"
)

type Stream struct {
	n int

	frequencyMap *hashmap.HashMap
}

type Element struct {
	Key   string
	Count uint64
}

func New(n int) *Stream {
	return &Stream{
		n: n,
		frequencyMap: hashmap.New(
			uintptr(len(common.Map)),
		),
	}
}

func (c *Stream) Keys() []Element {
	// NOTE:
	// 2^25 = 33554432
	// assume it's larger then a number of occurrences for the most frequent word
	freq := make([]*string, 2<<25)
	for kv := range c.frequencyMap.Iter() {
		wCopy := kv.Key.(string)
		counter := kv.Value.(*int64)
		freq[*counter] = &wCopy
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

	var i int64
	actual, _ := c.frequencyMap.GetOrInsert(word, &i)
	counter := (actual).(*int64)
	atomic.AddInt64(counter, 1)
}
