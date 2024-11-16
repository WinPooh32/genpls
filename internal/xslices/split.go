package xslices

import (
	"iter"
	"math"
)

// Split splits slice s to uniformly filled parts count of n.
func Split[Slice ~[]E, E any](s Slice, n int) iter.Seq[Slice] {
	if n < 1 {
		panic("cannot be less than 1")
	}

	return func(yield func(Slice) bool) {
		k := max(1, int(math.Round(float64(len(s))/float64(n))))

		for i := range n {
			start := i * k
			end := min((i+1)*k, len(s))

			if i == n-1 {
				end = len(s)
			}

			if end-start == 0 {
				return
			}

			if !yield(s[start:end:end]) {
				return
			}
		}
	}
}
