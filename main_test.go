package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"testing"

	"github.com/ngalaiko/words/count"
)

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

func Benchmark_read(b *testing.B) {
	files := []string{
		"./assets/test.txt",
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
