package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/ngalaiko/words/topk"
)

var filePath = flag.String("file", "", "path to input file")
var topN = flag.Int("n", 10, "top N words")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	tk := topk.New(*topN)
	err := fromFile(*filePath, tk)
	for _, key := range tk.Keys() {
		fmt.Printf("%d: %s\n", key.Count, key.Key)
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}

	if err != nil {
		log.Fatal(err)
	}
}

func fromFile(filepath string, tk *topk.Stream) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to read `%s`: %s", filepath, err)
	}

	if err := countWords(file, 2<<15-1, tk); err != nil {
		return err
	}

	return nil
}

func countWords(file *os.File, batchSize int64, tk *topk.Stream) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	var offset int64
	var done bool
	wg := &sync.WaitGroup{}
	for offset = 0; offset < stat.Size(); offset += batchSize {
		wg.Add(1)
		buff := make([]byte, batchSize)

		off, err := file.ReadAt(buff, offset)
		switch err {
		case io.EOF:
			done = true
		case nil:
		default:
			return err
		}

		go processBatch(buff[:off], tk, wg.Done)

		if done {
			break
		}
	}

	wg.Wait()

	return nil
}

// returns number of bytes processed
func processBatch(batch []byte, tk *topk.Stream, done func()) {
	defer done()

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

			tk.Insert(string(wordBuf[:wordPos]), 1)

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
	}
}

const maxUint = ^uint(0)

func processWords(wordsChan <-chan []byte, doneChan chan bool, tk *topk.Stream) {
	for word := range wordsChan {
		tk.Insert(string(word), 1)
	}

	close(doneChan)
}
