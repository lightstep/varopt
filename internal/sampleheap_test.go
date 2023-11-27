// Copyright 2019, LightStep Inc.

package internal_test

import (
	"container/heap"
	"math/rand"
	"testing"

	"github.com/lightstep/varopt/internal"
	"github.com/stretchr/testify/require"
)

type simpleHeap []float64

func (s *simpleHeap) Len() int {
	return len(*s)
}

func (s *simpleHeap) Swap(i, j int) {
	(*s)[i], (*s)[j] = (*s)[j], (*s)[i]
}

func (s *simpleHeap) Less(i, j int) bool {
	return (*s)[i] < (*s)[j]
}

func (s *simpleHeap) Push(x interface{}) {
	*s = append(*s, x.(float64))
}

func (s *simpleHeap) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

func TestLargeHeap(t *testing.T) {
	var L internal.SampleHeap[float64]
	var S simpleHeap

	for i := 0; i < 1e6; i++ {
		v := rand.NormFloat64()
		L.Push(internal.Vsample[float64]{
			Sample: v,
			Weight: v,
		})
		heap.Push(&S, v)
	}

	for len(S) > 0 {
		v1 := heap.Pop(&S)
		v2 := L.Pop().Weight

		require.Equal(t, v1, v2)
	}

	require.Equal(t, 0, len(L))
}
