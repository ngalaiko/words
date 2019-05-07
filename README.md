# Words count 

## Run: 
```go
go run main.go -file=/path/to/file
```

## Optimizations:

* Read file concurrently in batchs per `2^19-1` bytes [here](./main.go#L77)
* To get lowercase letter, add `32` to it's ASCII code [here](./main.go#L111)
* Use read optimized lock free map to count words [here](./count/stream.go#L25)
* To sort words in the end, add a word to a slice where it's index == number of occurrences, then interate
backwards and return first 10 words and it's indexes [here](./count/stream.go#L38)
* Count only most common words in the English language, because of the
[Law of large numbers](https://en.wikipedia.org/wiki/Law_of_large_numbers) [here](./count/stream.go#L59)
