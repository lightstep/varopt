package tdigest

import (
	"sort"
)

// Flat uses uniform probability density for each centroid.
type Flat struct {
	t *TDigest
}

var _ ProbabilityFunc = &Flat{}

func newFlatFunc(t *TDigest) *Flat {
	return &Flat{t: t}
}

func (f *Flat) Probability(value float64) float64 {
	digest := f.t.Digest
	sumw := f.t.SumWeight
	if value <= digest[0].Mean {
		return digest[0].Weight / (2 * sumw)
	}
	if value >= digest[len(digest)-1].Mean {
		return digest[len(digest)-1].Weight / (2 * sumw)
	}

	idx := sort.Search(len(digest), func(i int) bool {
		return value < digest[i].Mean
	})

	lower := value - digest[idx-1].Mean
	upper := digest[idx].Mean - value

	if lower > upper {
		return digest[idx-1].Weight / sumw
	}
	return digest[idx].Weight / sumw
}
