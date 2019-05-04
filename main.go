package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/dgryski/go-topk"
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

	if err := countWords(file, tk); err != nil {
		return err
	}

	return nil
}

const batchSize = 2<<15 - 1

func countWords(file *os.File, tk *topk.Stream) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}

	var offset int64
	var done bool
	for offset = 0; offset < stat.Size(); offset += batchSize {
		buff := make([]byte, batchSize)

		off, err := file.ReadAt(buff, offset)
		switch err {
		case io.EOF:
			done = true
		case nil:
		default:
			return err
		}

		processBatch(buff[:off], tk)

		if done {
			break
		}
	}

	return nil
}

// returns number of bytes processed
func processBatch(batch []byte, tk *topk.Stream) {
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
