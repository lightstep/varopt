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
