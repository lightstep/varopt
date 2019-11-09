// Copyright 2019, LightStep Inc.

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
