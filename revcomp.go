package main

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var waitGroup sync.WaitGroup
var cores = runtime.NumCPU()

// map of complementary nucleotides including some ambiguity codes
var compMap = map[string]string{"A": "T", "C": "G", "G": "C", "T": "A", "N": "N", "-": "-", "W": "W", "S": "S"}

// A structure to hold a sequence string (seq) and an index (idx). Used to keep a group of sequences in order during
// multithreaded operations using go routines
type indexedSequence struct {
	idx int
	seq string
}
// Creates an indexed sequence from the index (idx) and the sequence string (seq)
func newIndexedSequence(idx int, seq string) *indexedSequence {
	is := indexedSequence{idx, seq}
	return &is
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Error: Must supply at least one sequence as an argument")
		os.Exit(1)
	}

	// channel for results to come back on buffered for the number of cores
	results := make(chan *indexedSequence, cores)

	// assume each arg after the command (i = 0) is a sequence
	for i := 1; i < len(args); i += cores {

		// parallelize up to the number of cores available or sequences whichever is lesser
		for c := 0; c < cores && c+i < len(args); c++ {
			// expand the wait group for the go routine that will perform the work
			waitGroup.Add(1)

			// reverse complement the sequence according to the map and send results to the buffered result channel
			go ReverseComplement(newIndexedSequence(i+c, args[i+c]), compMap, results)
		}

		// wait for all the go routines to finish
		waitGroup.Wait()

		// collect the sequences in the result channel
		rcSequences := make([]*indexedSequence, 0, cores)
		for x := 0; x < cores && x+i < len(args); x++ {
			// these are probably out of order so we sort then when all are gathered
			rcSequences = append(rcSequences, <-results)
		}

		//sort the sequences into their index order using the supplied anonymous function
		sort.Slice(rcSequences, func(i, j int) bool { return rcSequences[i].idx < rcSequences[j].idx })

		//print out the sequences
		for x := 0; x < len(rcSequences); x++ {
			fmt.Println(rcSequences[x].seq)
		}
	}

	// close the channel
	close(results)
}

// Reverse complement a nucleotide sequence using the supplied complement map (compMap) and send the result to the
// results channel
func ReverseComplement(sequence *indexedSequence, compMap map[string]string, results chan *indexedSequence) {

	// defer telling the wait group we are done until the function completes
	defer waitGroup.Done()
	s := sequence.seq
	s = strings.ToUpper(s)
	s = strings.TrimSpace(s)

	var complementString string

	//reverse iterate the string (works because DNA is ASCII and not unicode otherwise I'd need runes)
	for i := len(s) - 1; i > -1; i-- {
		c := s[i : i+1]

		comp, isFound := compMap[c]
		if isFound {
			complementString += comp
		} else {
			fmt.Println("Error: No complementary nucleotide for character " + c)
			os.Exit(2)
		}
	}

	sequence.seq = complementString

	// send the result to the channel
	results <- sequence
}
