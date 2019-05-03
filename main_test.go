package main

import (
	"os"
	"testing"
)

func Benchmark_read_100_lines(b *testing.B) {
	file, err := os.Open("./assets/100lines.txt")
	if err != nil {
		b.Fatalf("failed to read `%s`: %s", *filePath, err)
	}

	wordsChan := make(chan string, 100000)

	go func() {
		for range wordsChan {
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		countWords(file, wordsChan)
	}
}
