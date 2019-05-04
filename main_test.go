package main

import (
	"testing"

	"github.com/dgryski/go-topk"
)

func Test(t *testing.T) {
	tk := topk.New(10)
	if err := fromFile("./assets/100000lines.txt", tk); err != nil {
		t.Fatal(err)
	}
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
