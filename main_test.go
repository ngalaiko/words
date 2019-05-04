package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"testing"

	"github.com/ngalaiko/words/topk"
)

func Test(t *testing.T) {
	file, err := ioutil.TempFile("assets", "test")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	wordsMap := map[string]int{}
	wordsMapCpy := map[string]int{}
	for i := 0; i < 100; i++ {
		length := rand.Intn(10) + 1
		word := randWord(length)
		wordsMap[word] = rand.Intn(100) + 1

		wordsMapCpy[word] = wordsMap[word]
	}

	for word := range wordsMap {
		if wordsMap[word] == 0 {
			continue
		}

		file.WriteString(word + "\n")
	}

	tk := topk.New(10)
	if err := fromFile(file.Name(), tk); err != nil {
		t.Fatal(err)
	}

	for _, key := range tk.Keys() {
		fmt.Printf("%s: got %d, expected %d\n", key.Key, key.Count, wordsMapCpy[key.Key])
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
				tk := topk.New(10)
				fromFile(filePath, tk)
			}
		})
	}
}
