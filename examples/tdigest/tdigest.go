// Package tdigest uses the t-digest Algorithm 1 to compute an ordered
// list of non-overlapping centroids, which can be used to estimate
// quantiles and probability density.
//
// What T-digest calls a "compression" paramter is called Quality here.
//
// See the 2019 paper here: https://arxiv.org/pdf/1902.04023.pdf
package tdigest

import (
	"fmt"
	"math"
	"sort"
)

type (
	ProbabilityFunc interface {
		Probability(value float64) float64
	}

	Centroid struct {
		Sum    float64 // Total sum of values
		Weight float64 // Total weight of values
		Mean   float64 // Sum / Weight
		Count  int     // Number of distinct values
	}

	// Quality is a quality parameter, must be > 0.
	//
	// Forcing the compression parameter to be an integer
	// simplifies the code, because each centroid / quantile has
	// k(i) == 1.
	Quality int

	// Item is a value/weight pair for input to the digest.
	Item struct {
		Value  float64
		Weight float64
	}

	// input is for sorting Items.
	Input []*Item

	// TDigest is computed using Algorithm 1 from the T-digest
	// paper.
	TDigest struct {
		Quality   Quality
		SumWeight float64
		Digest    []Centroid

		ProbabilityFunc
	}
)

var _ ProbabilityFunc = &TDigest{}

var (
	ErrEmptyDataSet  = fmt.Errorf("Empty data set")
	ErrInvalidWeight = fmt.Errorf("Negative or NaN weight")
	ErrInvalidInterp = fmt.Errorf("Unknown interpolation")
)

// The T-digest scaling function maps quantile to index, where
// 0 <= index < quality
func (quality Quality) digestK(q float64) float64 {
	return float64(quality) * (0.5 + math.Asin(2*q-1)/math.Pi)
}

// The T-digest inverse-scaling function maps index to quantile.
func (quality Quality) inverseK(k float64) float64 {
	if k > float64(quality) {
		return 1
	}
	return 0.5 * (math.Sin(math.Pi*(k/float64(quality)-0.5)) + 1)
}

func New(quality Quality) *TDigest {
	return &TDigest{
		Quality: quality,
		Digest:  make([]Centroid, 0, quality),
	}
}

func (t *TDigest) Compute(in Input) error {
	t.Digest = t.Digest[:0]

	if len(in) == 0 {
		return ErrEmptyDataSet
	}

	// Compute the total weight, check for invalid weights.
	// combine weights for equal values.

	sumWeight := 0.0

	for _, it := range in {
		we := it.Weight
		if we < 0 || math.IsNaN(we) || math.IsInf(we, +1) {
			return ErrInvalidWeight
		}

		sumWeight += we
	}

	// Both of the following loops require sorted data.
	sort.Sort(&in)

	outIndex := 0
	for i := 0; i < len(in); i++ {
		va := in[i].Value
		we := in[i].Weight

		if we == 0 {
			continue
		}

		if outIndex != 0 && i != 0 && in[i-1].Value == va {
			in[outIndex-1].Weight += we
			continue
		}

		in[outIndex].Value = va
		in[outIndex].Weight = we
		outIndex++
	}

	in = in[0:outIndex]

	// T-digest's Algorithm 1.  The step above to de-duplicate
	// values ensures that each bucket has a non-zero width.
	digest := t.Digest

	qleft := 0.0
	qlimit := t.Quality.inverseK(1)

	current := Centroid{}

	for pos := 0; pos < len(in); pos++ {
		vpos := in[pos].Value
		wpos := in[pos].Weight

		q := qleft + (current.Weight+wpos)/sumWeight

		if q <= qlimit {
			current.Sum += vpos * wpos
			current.Weight += wpos
			current.Count++
			continue
		}

		if current.Count > 0 {
			digest = append(digest, current)
		}
		qleft += current.Weight / sumWeight

		qlimit = t.Quality.inverseK(t.Quality.digestK(qleft) + 1)
		current.Sum = vpos * wpos
		current.Weight = wpos
		current.Count = 1
	}
	digest = append(digest, current)

	t.Digest = digest
	t.SumWeight = sumWeight
	t.ProbabilityFunc = newFlatFunc(t)

	return nil
}

func (in *Input) Len() int {
	return len(*in)
}

func (in *Input) Swap(i, j int) {
	(*in)[i], (*in)[j] = (*in)[j], (*in)[i]
}

func (in *Input) Less(i, j int) bool {
	return (*in)[i].Value < (*in)[j].Value
}
