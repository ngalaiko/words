package main

import (
	"os"
	"testing"
)

func Benchmark_read(b *testing.B) {
	files := []string{
		//"./assets/10lines.txt",
		//"./assets/100lines.txt",
		"./assets/1000lines.txt",
	}

	for _, filePath := range files {
		b.Run(filePath, func(b *testing.B) {
			file, err := os.Open(filePath)
			if err != nil {
				b.Fatalf("failed to read `%s`: %s", filePath, err)
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
		})
	}
}
