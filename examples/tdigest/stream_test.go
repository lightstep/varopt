package tdigest_test

import (
	"math"
	"math/rand"
	"sort"
	"testing"

	"github.com/lightstep/varopt/examples/tdigest"
	"github.com/stretchr/testify/require"
)

func TestStreamNormal(t *testing.T) {
	const (
		mean   = 10
		stddev = 10
	)

	testStreamDigest(t,
		[]float64{.1, .25, .5, .75, .9, .99, .999},
		[]float64{.5, .50, .1, .50, .5, .05, .050},
		func(rnd *rand.Rand) float64 {
			return mean + stddev*rnd.NormFloat64()
		})
}

func TestStreamExponential(t *testing.T) {
	testStreamDigest(t,
		[]float64{.1, .25, .5, .75, .9, .99, .999},
		[]float64{.5, .50, 1., 1.5, 1., .20, .050},
		func(rnd *rand.Rand) float64 {
			return rnd.ExpFloat64()
		})
}

func TestStreamAlternating(t *testing.T) {
	count := 0
	testStreamDigest(t,
		// Quantiles <0.66 == -1
		// Quantiles >=0.66 == +1
		[]float64{.1, .5, .66, .67, .7, .8, .87, .9},
		// This performs poorly, but it's hard to show
		// concisely with this test framework.  At the 90th
		// percentile, we finally see a +1
		[]float64{10, 50, 66, 67, 70, 80, 21, 24},
		func(rnd *rand.Rand) float64 {
			c := count
			count++
			count = count % 3
			switch c {
			case 0, 2:
				return -1
			default:
				return +1
			}
		})
}

func TestStreamLinear(t *testing.T) {
	count := 0
	testStreamDigest(t,
		[]float64{.1, .25, .5, .75, .9, .99, .999},
		[]float64{.5, .5, .5, 1., .5, .5, .5},
		func(rnd *rand.Rand) float64 {
			c := count
			count++
			return float64(c)
		})
}

func TestStreamUniform(t *testing.T) {
	testStreamDigest(t,
		[]float64{.1, .25, .5, .75, .9, .99, .999},
		[]float64{.5, .5, .5, 1., .5, .5, .5},
		func(rnd *rand.Rand) float64 {
			return rnd.Float64()
		})
}

func testStreamDigest(t *testing.T, quantiles, tolerance []float64, f func(rnd *rand.Rand) float64) {
	const (
		quality     = 5000
		sampleSize  = 5000
		windowSize  = 25000
		streamCount = 1000000
	)

	rnd := rand.New(rand.NewSource(31181))
	correct := &CorrectQuantile{}

	stream := tdigest.NewStream(tdigest.NewConfig(quality, sampleSize, sampleSize, windowSize), rnd)

	for i := 0; i < streamCount; i++ {
		value := f(rnd)
		correct.Add(value)
		err := stream.Add(value)
		if err != nil {
			t.Error("Stream add error", err)
		}
	}

	for i, q := range quantiles {
		require.GreaterOrEqual(t, tolerance[i], math.Abs(correct.Distance(stream, q)),
			"at quantile=%v", q)
	}
}

type Quantiler interface {
	Quantile(float64) float64
}

type CorrectQuantile struct {
	values []float64
	sorted bool
}

// Distance returns quantile distance in percent units.
func (l *CorrectQuantile) Distance(qq Quantiler, quant float64) float64 {
	value := qq.Quantile(quant)
	actual := l.LookupQuantile(value)
	return 100 * (actual - quant)
}

func (l *CorrectQuantile) Add(f float64) {
	l.values = append(l.values, f)
	l.sorted = false
}

func (l *CorrectQuantile) Quantile(f float64) float64 {
	if len(l.values) == 0 {
		return math.NaN()
	}
	if !l.sorted {
		sort.Float64s(l.values)
		l.sorted = true
	}
	quantileLocation := float64(len(l.values)) * f
	if quantileLocation <= 0 {
		return l.values[0]
	}
	if quantileLocation >= float64(len(l.values)-1) {
		return l.values[len(l.values)-1]
	}
	beforeIndex := int(math.Floor(quantileLocation))
	afterIndex := beforeIndex + 1
	delta := l.values[afterIndex] - l.values[beforeIndex]
	if delta == 0 {
		return l.values[beforeIndex]
	}
	return l.values[beforeIndex] + delta*(quantileLocation-float64(beforeIndex))
}

func (l *CorrectQuantile) LookupQuantile(value float64) float64 {
	if !l.sorted {
		sort.Float64s(l.values)
		l.sorted = true
	}

	idx := sort.Search(len(l.values), func(i int) bool {
		return l.values[i] >= value
	})

	if idx == 0 {
		return 0
	}

	if idx == len(l.values) {
		return 1
	}

	above := l.values[idx]
	below := l.values[idx-1]
	diff := above - below
	if diff == 0 {
		panic("impossible")
	}
	return (float64(idx) + (value-below)/diff) / float64(len(l.values)-1)
}
