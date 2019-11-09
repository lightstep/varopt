// Copyright 2019, LightStep Inc.
//
// The benchmark results point to a performance drop when the
// largeHeap starts to be used because of interface conversions in and
// out of the heap, primarily due to the heap interface.  This
// suggests room for improvement by avoiding the built-in heap.

/*
BenchmarkAdd_Norm_100-8       	37540165	        32.1 ns/op	       8 B/op	       0 allocs/op
BenchmarkAdd_Norm_10000-8     	39850280	        30.6 ns/op	       8 B/op	       0 allocs/op
BenchmarkAdd_Norm_1000000-8   	 7958835	       183 ns/op	      52 B/op	       0 allocs/op
BenchmarkAdd_Exp_100-8        	41565934	        28.5 ns/op	       8 B/op	       0 allocs/op
BenchmarkAdd_Exp_10000-8      	43622184	        29.2 ns/op	       8 B/op	       0 allocs/op
BenchmarkAdd_Exp_1000000-8    	 8103663	       220 ns/op	      55 B/op	       0 allocs/op
*/

package varopt_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/lightstep/varopt"
)

func normValue(rnd *rand.Rand) float64 {
	return 1 + math.Abs(rnd.NormFloat64())
}

func expValue(rnd *rand.Rand) float64 {
	return rnd.ExpFloat64()
}

func BenchmarkAdd_Norm_100(b *testing.B) {
	benchmarkAdd(b, 100, normValue)
}

func BenchmarkAdd_Norm_10000(b *testing.B) {
	benchmarkAdd(b, 10000, normValue)
}

func BenchmarkAdd_Norm_1000000(b *testing.B) {
	benchmarkAdd(b, 1000000, normValue)
}

func BenchmarkAdd_Exp_100(b *testing.B) {
	benchmarkAdd(b, 100, expValue)
}

func BenchmarkAdd_Exp_10000(b *testing.B) {
	benchmarkAdd(b, 10000, expValue)
}

func BenchmarkAdd_Exp_1000000(b *testing.B) {
	benchmarkAdd(b, 1000000, expValue)
}

func benchmarkAdd(b *testing.B, size int, f func(rnd *rand.Rand) float64) {
	b.ReportAllocs()
	rnd := rand.New(rand.NewSource(3331))
	v := varopt.New(size, rnd)
	weights := make([]float64, b.N)
	for i := 0; i < b.N; i++ {
		weights[i] = f(rnd)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		v.Add(nil, weights[i])
	}
}
