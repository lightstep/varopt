// Copyright 2019, LightStep Inc.

package varopt_test

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/lightstep/varopt"
)

type curve struct {
	color  string
	mean   float64
	stddev float64
}

type testPoint struct {
	color  int
	xvalue float64
}

var colors = []curve{
	{color: "red", mean: 10, stddev: 15},
	{color: "green", mean: 30, stddev: 10},
	{color: "blue", mean: 50, stddev: 20},
}

// This example shows how to use Varopt sampling to estimate
// frequencies with the use of inverse probability weights.  The use
// of inverse probability creates a uniform expected value, in this of
// the number of sample points per second.
//
// While the number of expected points per second is uniform, the
// output sample weights are expected to match the original
// frequencies.
func ExampleVaropt_GetOriginalWeight() {
	// Number of points.
	const totalCount = 1e6

	// Relative size of the sample.
	const sampleRatio = 0.01

	// Ensure this test is deterministic.
	rnd := rand.New(rand.NewSource(104729))

	// Construct a timeseries consisting of three colored signals,
	// for x=0 to x=60 seconds.
	var points []testPoint

	// origCounts stores the original signals at second granularity.
	origCounts := make([][]int, len(colors))
	for i := range colors {
		origCounts[i] = make([]int, 60)
	}

	// Construct the signals by choosing a random color, then
	// using its Gaussian to compute a timestamp.
	for len(points) < totalCount {
		choose := rnd.Intn(len(colors))
		series := colors[choose]
		xvalue := rnd.NormFloat64()*series.stddev + series.mean

		if xvalue < 0 || xvalue > 60 {
			continue
		}
		origCounts[choose][int(math.Floor(xvalue))]++
		points = append(points, testPoint{
			color:  choose,
			xvalue: xvalue,
		})
	}

	// Compute the total number of points per second.  This will be
	// used to establish the per-second probability.
	xcount := make([]int, 60)
	for _, point := range points {
		xcount[int(math.Floor(point.xvalue))]++
	}

	// Compute the sample with using the inverse probability as a
	// weight.  This ensures a uniform distribution of points in each
	// second.
	sampleSize := int(sampleRatio * float64(totalCount))
	sampler := varopt.New[testPoint](sampleSize, rnd)
	for _, point := range points {
		second := int(math.Floor(point.xvalue))
		prob := float64(xcount[second]) / float64(totalCount)
		sampler.Add(point, 1/prob)
	}

	// sampleCounts stores the reconstructed signals.
	sampleCounts := make([][]float64, len(colors))
	for i := range colors {
		sampleCounts[i] = make([]float64, 60)
	}

	// pointCounts stores the number of points per second.
	pointCounts := make([]int, 60)

	// Reconstruct the signals using the output sample weights.
	// The effective count of each sample point is its output
	// weight divided by its original weight.
	for i := 0; i < sampler.Size(); i++ {
		point, weight := sampler.Get(i)
		origWeight := sampler.GetOriginalWeight(i)
		second := int(math.Floor(point.xvalue))
		sampleCounts[point.color][second] += (weight / origWeight)
		pointCounts[second]++
	}

	// Compute standard deviation of sample points per second.
	sum := 0.0
	mean := float64(sampleSize) / 60
	for s := 0; s < 60; s++ {
		e := float64(pointCounts[s]) - mean
		sum += e * e
	}
	stddev := math.Sqrt(sum / (60 - 1))

	fmt.Printf("Samples per second mean %.2f\n", mean)
	fmt.Printf("Samples per second standard deviation %.2f\n", stddev)

	// Compute mean absolute percentage error between sampleCounts
	// and origCounts for each signal.
	for c := range colors {
		mae := 0.0
		for s := 0; s < 60; s++ {
			mae += math.Abs(sampleCounts[c][s]-float64(origCounts[c][s])) / float64(origCounts[c][s])
		}
		mae /= 60
		fmt.Printf("Mean absolute percentage error (%s) = %.2f%%\n", colors[c].color, mae*100)
	}

	// Output:
	// Samples per second mean 166.67
	// Samples per second standard deviation 13.75
	// Mean absolute percentage error (red) = 25.16%
	// Mean absolute percentage error (green) = 14.30%
	// Mean absolute percentage error (blue) = 14.23%
}
