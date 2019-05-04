package topk

import (
	"bytes"
	"container/heap"
	"encoding/gob"
	"sort"
	"sync"

	"github.com/dgryski/go-sip13"
)

// Element is a TopK item
type Element struct {
	Key   string
	Count int
	Error int
}

type elementsByCountDescending []Element

func (elts elementsByCountDescending) Len() int { return len(elts) }
func (elts elementsByCountDescending) Less(i, j int) bool {
	return (elts[i].Count > elts[j].Count) || (elts[i].Count == elts[j].Count && elts[i].Key < elts[j].Key)
}
func (elts elementsByCountDescending) Swap(i, j int) { elts[i], elts[j] = elts[j], elts[i] }

type keys struct {
	mGuard    *sync.RWMutex
	m         map[string]int
	elts      []Element
	eltsGuard *sync.RWMutex
}

// Implement the container/heap interface

func (tk *keys) Len() int { return len(tk.elts) }
func (tk *keys) Less(i, j int) bool {
	tk.eltsGuard.RLock()
	defer tk.eltsGuard.RUnlock()
	return (tk.elts[i].Count < tk.elts[j].Count) || (tk.elts[i].Count == tk.elts[j].Count && tk.elts[i].Error > tk.elts[j].Error)
}
func (tk *keys) Swap(i, j int) {
	tk.eltsGuard.Lock()
	tk.elts[i], tk.elts[j] = tk.elts[j], tk.elts[i]
	iKey := tk.elts[i].Key
	jKey := tk.elts[j].Key
	tk.eltsGuard.Unlock()

	tk.mGuard.Lock()
	tk.m[iKey] = i
	tk.m[jKey] = j
	tk.mGuard.Unlock()
}

func (tk *keys) Push(x interface{}) {
	e := x.(Element)

	tk.mGuard.Lock()
	tk.m[e.Key] = len(tk.elts)
	tk.mGuard.Unlock()

	tk.eltsGuard.Lock()
	tk.elts = append(tk.elts, e)
	tk.eltsGuard.Unlock()
}

func (tk *keys) Pop() interface{} {
	var e Element
	e, tk.elts = tk.elts[len(tk.elts)-1], tk.elts[:len(tk.elts)-1]

	delete(tk.m, e.Key)

	return e
}

// Stream calculates the TopK elements for a stream
type Stream struct {
	n int
	k keys

	alphas       []int
	alphasGruard *sync.RWMutex
}

// New returns a Stream estimating the top n most frequent elements
func New(n int) *Stream {
	return &Stream{
		n: n,
		k: keys{
			m: make(map[string]int), mGuard: &sync.RWMutex{},
			elts: make([]Element, 0, n), eltsGuard: &sync.RWMutex{},
		},
		alphas:       make([]int, n*6), // 6 is the multiplicative constant from the paper
		alphasGruard: &sync.RWMutex{},
	}
}

func reduce(x uint64, n int) uint32 {
	return uint32(uint64(uint32(x)) * uint64(n) >> 32)
}

// Insert adds an element to the stream to be tracked
// It returns an estimation for the just inserted element
func (s *Stream) Insert(x string, count int) Element {

	xhash := reduce(sip13.Sum64Str(0, 0, x), len(s.alphas))

	// are we tracking this element?
	s.k.mGuard.RLock()
	idx, ok := s.k.m[x]
	s.k.mGuard.RUnlock()
	if ok {
		s.k.eltsGuard.Lock()
		s.k.elts[idx].Count += count
		e := s.k.elts[idx]
		s.k.eltsGuard.Unlock()
		heap.Fix(&s.k, idx)
		return e
	}

	// can we track more elements?
	if len(s.k.elts) < s.n {
		// there is free space
		e := Element{Key: x, Count: count}
		heap.Push(&s.k, e)
		return e
	}

	s.alphasGruard.RLock()
	alphasXHash := s.alphas[xhash]
	s.alphasGruard.RUnlock()

	s.k.eltsGuard.RLock()
	eltsCount := s.k.elts[0].Count
	s.k.eltsGuard.RUnlock()

	if alphasXHash+count < eltsCount {
		e := Element{
			Key:   x,
			Error: alphasXHash,
			Count: alphasXHash + count,
		}
		s.alphasGruard.Lock()
		s.alphas[xhash] += count
		alphasXHash += count
		s.alphasGruard.Unlock()
		return e
	}

	// replace the current minimum element
	s.k.eltsGuard.RLock()
	minKey := s.k.elts[0].Key
	s.k.eltsGuard.RUnlock()

	mkhash := reduce(sip13.Sum64Str(0, 0, minKey), len(s.alphas))

	s.alphasGruard.Lock()
	s.alphas[mkhash] = eltsCount
	s.alphasGruard.Unlock()

	e := Element{
		Key:   x,
		Error: alphasXHash,
		Count: alphasXHash + count,
	}
	s.k.eltsGuard.Lock()
	s.k.elts[0] = e
	s.k.eltsGuard.Unlock()

	s.k.mGuard.Lock()
	// we're not longer monitoring minKey
	delete(s.k.m, minKey)
	// but 'x' is as array position 0
	s.k.m[x] = 0
	s.k.mGuard.Unlock()

	heap.Fix(&s.k, 0)
	return e
}

// Keys returns the current estimates for the most frequent elements
func (s *Stream) Keys() []Element {
	elts := append([]Element(nil), s.k.elts...)
	sort.Sort(elementsByCountDescending(elts))
	return elts
}

// Estimate returns an estimate for the item x
func (s *Stream) Estimate(x string) Element {
	xhash := reduce(sip13.Sum64Str(0, 0, x), len(s.alphas))

	// are we tracking this element?
	if idx, ok := s.k.m[x]; ok {
		e := s.k.elts[idx]
		return e
	}
	count := s.alphas[xhash]
	e := Element{
		Key:   x,
		Error: count,
		Count: count,
	}
	return e
}

func (s *Stream) GobEncode() ([]byte, error) {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s.n); err != nil {
		return nil, err
	}
	if err := enc.Encode(s.k.m); err != nil {
		return nil, err
	}
	if err := enc.Encode(s.k.elts); err != nil {
		return nil, err
	}
	if err := enc.Encode(s.alphas); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Stream) GobDecode(b []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	if err := dec.Decode(&s.n); err != nil {
		return err
	}
	if err := dec.Decode(&s.k.m); err != nil {
		return err
	}
	if err := dec.Decode(&s.k.elts); err != nil {
		return err
	}
	if err := dec.Decode(&s.alphas); err != nil {
		return err
	}
	return nil
}
