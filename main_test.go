package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ngalaiko/words/count"
)

func Test_processBatch(t *testing.T) {
	batch := `
the
(of <of>
get) get get
With with with-wiTH
that THAT that that (ThAt)
`
	tk := count.New(10)
	processBatch([]byte(batch), 4, tk)

	resMap := map[string]int{}
	for _, key := range tk.Keys() {
		resMap[key.Key] = int(key.Count)
	}

	assert.Equal(t, 1, resMap["the"])
	assert.Equal(t, 2, resMap["of"])
	assert.Equal(t, 3, resMap["get"])
	assert.Equal(t, 4, resMap["with"])
	assert.Equal(t, 5, resMap["that"])
}

func Test(t *testing.T) {
	file, err := ioutil.TempFile("assets", "test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	wordsMap := map[string]uint64{}
	wordsMapCpy := map[string]uint64{}
	for i := 0; i < 100; i++ {
		length := rand.Intn(10) + 1
		word := randWord(length)
		wordsMap[word] = uint64(rand.Intn(32) + 1)

		wordsMapCpy[word] = wordsMap[word]
	}

	for len(wordsMap) != 0 {
		var word string
		for w := range wordsMap {
			word = w
		}

		file.WriteString(word + " ")
		if rand.Float64() >= 0.5 {
			file.WriteString("\n")
		}

		if wordsMap[word] == 1 {
			delete(wordsMap, word)
		} else {
			wordsMap[word]--
		}
	}

	tk := count.New(10)
	if err := fromFile(file.Name(), 100, tk); err != nil {
		t.Fatal(err)
	}

	for _, e := range tk.Keys() {
		if uint64(e.Count) != wordsMapCpy[e.Key] {
			fmt.Printf("%s: got %d, expected %d\n", e.Key, e.Count, wordsMapCpy[e.Key])
		}
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randWord(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]

	}
	return string(b)

}

func Benchmark__length(b *testing.B) {
	files := []string{
		"./assets/10lines.txt",
		"./assets/100lines.txt",
		"./assets/1000lines.txt",
		"./assets/10000lines.txt",
		"./assets/100000lines.txt",
	}

	for _, filePath := range files {
		b.Run(filePath, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				tk := count.New(10)
				fromFile(filePath, 2<<15-1, tk)
			}
		})
	}
}

func Benchmark_buffer(b *testing.B) {
	bufSizes := []int64{
		2<<16 - 1,
		2<<17 - 1,
		2<<18 - 1,
		2<<19 - 1,
		2<<20 - 1,
		2<<21 - 1,
	}

	for _, size := range bufSizes {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				tk := count.New(10)
				fromFile("./assets/1000000lines.txt", size, tk)
			}
		})
	}
}
