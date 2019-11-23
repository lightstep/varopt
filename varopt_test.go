// Copyright 2019, LightStep Inc.

package varopt_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/lightstep/varopt"
	"github.com/lightstep/varopt/simple"
	"github.com/stretchr/testify/require"
)

// There are 2 unequal sizes of simple block
// There are odd and even numbers, in equal amount
// There are last-digits 0-9 in equal amount
//
// We will test the mean is correct and, because unbiased, also the
// odd/even and last-digit-0-9 groupings will be balanced.
const (
	numBlocks      = 100
	popSize        = 1e7
	sampleProb     = 0.001
	sampleSize int = popSize * sampleProb

	epsilon = 0.08
)

func TestUnbiased(t *testing.T) {
	var (
		// Ratio of big blocks to small blocks
		bigBlockRatios = []float64{0.1, 0.3, 0.5, 0.7, 0.9, 1.0}
		// Ratio of big block size to small block size
		bigSizeRatios = []float64{0.1, 0.2, 0.4}
	)

	for _, bbr := range bigBlockRatios {
		for _, bsr := range bigSizeRatios {
			testUnbiased(t, bbr, bsr)
		}
	}
}

func testUnbiased(t *testing.T, bbr, bsr float64) {
	var (
		numBig   = int(numBlocks * bbr)
		numSmall = numBlocks - numBig

		factor = float64(numBig)/bsr + float64(numSmall)

		smallSize = int(popSize / factor)
		bigSize   = int(float64(smallSize) / bsr)

		extra = popSize - bigSize*numBig - smallSize*numSmall
	)

	population := make([]varopt.Sample, popSize)

	psum := 0.0

	for i := range population {
		population[i] = i
		psum += float64(i)
	}

	// Note: We're leaving the data unsorted to prove lack of bias
	// rand.Shuffle(len(population), func(i, j int) {
	// 	population[i], population[j] = population[j], population[i]
	// })

	smallBlocks := make([][]varopt.Sample, numSmall)
	bigBlocks := make([][]varopt.Sample, numBig)

	for i := 0; i < numSmall; i++ {
		smallBlocks[i] = make([]varopt.Sample, smallSize)
	}
	for i := 0; i < numBig; i++ {
		if i == 0 {
			bigBlocks[0] = make([]varopt.Sample, bigSize+extra)
		} else {
			bigBlocks[i] = make([]varopt.Sample, bigSize)
		}
	}

	pos := 0
	for i := 0; i < numSmall; i++ {
		for j := 0; j < len(smallBlocks[i]); j++ {
			smallBlocks[i][j] = population[pos]
			pos++
		}
	}
	for i := 0; i < numBig; i++ {
		for j := 0; j < len(bigBlocks[i]); j++ {
			bigBlocks[i][j] = population[pos]
			pos++
		}
	}
	require.Equal(t, len(population), pos)

	maxDiff := 0.0
	rnd := rand.New(rand.NewSource(98887))

	func(allBlockLists ...[][][]varopt.Sample) {
		for _, blockLists := range allBlockLists {
			vsample := varopt.New(sampleSize, rnd)

			for _, blockList := range blockLists {
				for _, block := range blockList {
					ss := simple.New(sampleSize, rnd)

					for _, s := range block {
						ss.Add(s)
					}

					weight := ss.Weight()
					for i := 0; i < ss.Size(); i++ {
						vsample.Add(ss.Get(i), weight)
					}
				}
			}

			vsum := 0.0
			odd := 0.0
			even := 0.0

			for i := 0; i < vsample.Size(); i++ {
				v, w := vsample.Get(i)
				vi := v.(int)
				if vi%2 == 0 {
					even++
				} else {
					odd++
				}

				vsum += w * float64(vi)
			}

			diff := math.Abs(vsum-psum) / psum
			maxDiff = math.Max(maxDiff, diff)

			require.InEpsilon(t, vsum, psum, epsilon)
			require.InEpsilon(t, odd, even, epsilon)
		}
	}(
		[][][]varopt.Sample{bigBlocks, smallBlocks},
		[][][]varopt.Sample{smallBlocks, bigBlocks},
	)
}

func TestInvalidWeight(t *testing.T) {
	rnd := rand.New(rand.NewSource(98887))
	v := varopt.New(1, rnd)

	_, err := v.Add(nil, math.NaN())
	require.Equal(t, err, varopt.ErrInvalidWeight)

	_, err = v.Add(nil, -1)
	require.Equal(t, err, varopt.ErrInvalidWeight)

	_, err = v.Add(nil, 0)
	require.Equal(t, err, varopt.ErrInvalidWeight)
}

func TestReset(t *testing.T) {
	const capacity = 10
	const insert = 100
	rnd := rand.New(rand.NewSource(98887))
	v := varopt.New(capacity, rnd)

	sum := 0.
	for i := 1.; i <= insert; i++ {
		v.Add(nil, i)
		sum += i
	}

	require.Equal(t, capacity, v.Size())
	require.Equal(t, insert, v.TotalCount())
	require.Equal(t, sum, v.TotalWeight())
	require.Less(t, 0., v.Tau())

	v.Reset()

	require.Equal(t, 0, v.Size())
	require.Equal(t, 0, v.TotalCount())
	require.Equal(t, 0., v.TotalWeight())
	require.Equal(t, 0., v.Tau())
}

func TestEject(t *testing.T) {
	const capacity = 100
	const rounds = 10000
	const maxvalue = 10000

	entries := make([]int, capacity+1)
	freelist := make([]*int, capacity+1)

	for i := range entries {
		freelist[i] = &entries[i]
	}

	// Make two deterministically equal samplers
	rnd1 := rand.New(rand.NewSource(98887))
	rnd2 := rand.New(rand.NewSource(98887))
	vsrc := rand.New(rand.NewSource(98887))

	expected := varopt.New(capacity, rnd1)
	ejector := varopt.New(capacity, rnd2)

	for i := 0; i < rounds; i++ {
		value := vsrc.Intn(maxvalue)
		weight := vsrc.ExpFloat64()

		_, _ = expected.Add(&value, weight)

		lastitem := len(freelist) - 1
		item := freelist[lastitem]
		freelist = freelist[:lastitem]

		*item = value
		eject, _ := ejector.Add(item, weight)

		if eject != nil {
			freelist = append(freelist, eject.(*int))
		}
	}

	require.Equal(t, expected.Size(), ejector.Size())
	require.Equal(t, expected.TotalCount(), ejector.TotalCount())
	require.Equal(t, expected.TotalWeight(), ejector.TotalWeight())
	require.Equal(t, expected.Tau(), ejector.Tau())

	for i := 0; i < capacity; i++ {
		expectItem, expectWeight := expected.Get(i)
		ejectItem, ejectWeight := expected.Get(i)

		require.Equal(t, *expectItem.(*int), *ejectItem.(*int))
		require.Equal(t, expectWeight, ejectWeight)
	}
}
