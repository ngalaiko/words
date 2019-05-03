package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/dgryski/go-topk"
)

var filePath = flag.String("file", "", "path to input file")

func main() {
	flag.Parse()

	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("failed to read `%s`: %s", *filePath, err)
	}

	wordsChan := make(chan string)
	doneChan := make(chan bool)

	go processWords(wordsChan, doneChan, 10)

	if err := countWords(file, wordsChan); err != nil {
		log.Fatal(err)
	}

	close(wordsChan)

	<-doneChan
}

const buffSize = 2048

func countWords(file *os.File, wordsChan chan<- string) error {
	buff := make([]byte, buffSize)

	var offset int64
	var done bool
	for {
		off, err := file.ReadAt(buff, offset)
		switch err {
		case io.EOF:
			done = true
		case nil:
		default:
			return err
		}

		off, err = processBatch(buff[:off], wordsChan)
		if err != nil {
			return err
		}

		if done {
			return nil
		}

		offset += int64(off)
	}
}

// returns number of bytes processed
func processBatch(batch []byte, wordsChan chan<- string) (int, error) {
	processed := 0

	wordBuf := make([]byte, 16)
	wordPos := 0

	// TODO: there might be a case when a word is splitted by a buffered read

	for _, c := range batch {
		skipWord := false
		switch {
		// 10 is a `LF` char for at the end of file
		case c == ' ', c == 10:
			if skipWord {
				continue
			}

			if wordPos == 0 {
				continue
			}

			wordsChan <- string(wordBuf[:wordPos])

			wordPos = 0
		case c < 'A' || c > 'z', c > 'Z' && c < 'a':
			wordPos = 0
		default:
			if wordPos >= cap(wordBuf) {
				// skip too long words
				skipWord = true
				continue
			}
			wordBuf[wordPos] = c
			wordPos++
		}

		processed++
	}

	return processed, nil
}

const maxUint = ^uint(0)

func processWords(wordsChan <-chan string, doneChan chan bool, topN int) {
	tk := topk.New(topN)
	for word := range wordsChan {
		tk.Insert(string(word), 1)
	}

	for _, key := range tk.Keys() {
		fmt.Printf("%d: %s\n", key.Count, key.Key)
	}

	close(doneChan)
}
