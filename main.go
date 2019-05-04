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
	err := fromFile(*filePath, 2<<15-1, tk)
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
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat `%s`: %s", filepath, err)
	}
	_ = file.Close()

	wg := &errgroup.Group{}
	for i := int64(0); i < info.Size()/batchSize+1; i++ {
		i := i
		wg.Go(func() error {
			file, err := os.Open(filepath)
			if err != nil {
				return fmt.Errorf("failed to read `%s`: %s", filepath, err)
			}

			buff := make([]byte, batchSize)

			off, err := file.ReadAt(buff, batchSize*i)
			switch err {
			case nil:
			case io.EOF:
				return nil
			default:
				return err
			}

			processBatch(buff[:off], tk)

			return nil
		})
	}

	return wg.Wait()
}

// returns number of bytes processed
func processBatch(batch []byte, tk *count.Stream) {
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

			word := string(wordBuf[:wordPos])
			tk.Insert(word)

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
