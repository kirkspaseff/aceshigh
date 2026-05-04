// Package stats tracks counters during an ingestion run for end-of-run
// reporting. All operations are safe for use from a single goroutine;
// add a sync.Mutex if you parallelize loading later.
package stats

import (
	"fmt"
	"io"
	"sort"
)

type Counter struct {
	counts map[string]int
}

func New() *Counter {
	return &Counter{counts: make(map[string]int)}
}

func (c *Counter) Inc(key string) {
	c.counts[key]++
}

func (c *Counter) Add(key string, n int) {
	c.counts[key] += n
}

func (c *Counter) Get(key string) int {
	return c.counts[key]
}

// WriteReport writes a sorted summary of all counters to w.
func (c *Counter) WriteReport(w io.Writer) {
	keys := make([]string, 0, len(c.counts))
	for k := range c.counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "  %-40s %d\n", k, c.counts[k])
	}
}

