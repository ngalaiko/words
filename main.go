package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"golang.org/x/sync/errgroup"

	"github.com/ngalaiko/words/count"
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

	tk := count.New(*topN)
	err := fromFile(*filePath, 2<<19-1, tk)
	for _, e := range tk.Keys() {
		fmt.Printf("%d: %s\n", e.Count, e.Key)
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

func fromFile(filepath string, batchSize int64, tk *count.Stream) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to read `%s`: %s", filepath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat `%s`: %s", filepath, err)
	}

	all := info.Size() / batchSize
	wg := &errgroup.Group{}
	for i := int64(0); i < all; i++ {
		i := i
		// NOTE: read concurrently and process in batch
		wg.Go(func() error {

			buff := make([]byte, batchSize)

			off, err := file.ReadAt(buff, batchSize*i)
			switch err {
			case nil:
			case io.EOF:
				return nil
			default:
				return err
			}

			processBatch(buff[:off], maxLen, tk)

			return nil
		})
	}

	return wg.Wait()
}

const maxLen = 4

// returns number of bytes processed
func processBatch(batch []byte, maxLen int, tk *count.Stream) {
	wordBuf := make([]byte, maxLen)
	wordPos := 0

	// TODO: there is a case when a word is splitted by a buffered read
	// NOTE: I don't care

	for _, c := range batch {
		switch {
		case c >= 'A' && c <= 'Z':
			c += 32
			fallthrough
		case c >= 'a' && c <= 'z':
			if wordPos == maxLen {
				continue
			}
			wordBuf[wordPos] = c
			wordPos++
		default:
			if wordPos == 0 {
				continue
			}

			tk.Insert(string(wordBuf[:wordPos]))

			wordPos = 0
		}
	}

	if wordPos > 0 && wordPos <= maxLen {
		tk.Insert(string(wordBuf[:wordPos]))
	}
}
