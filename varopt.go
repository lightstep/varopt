// Copyright 2019, LightStep Inc.

package varopt

import (
	"fmt"
	"math/rand"
)

// Varopt implements the algorithm from Stream sampling for
// variance-optimal estimation of subset sums Edith Cohen, Nick
// Duffield, Haim Kaplan, Carsten Lund, Mikkel Thorup 2008
//
// https://arxiv.org/pdf/0803.0473.pdf
type Varopt struct {
	// Random number generator
	rnd *rand.Rand

	// Large-weight items stored in a min-heap.
	L largeHeap

	// Light-weight items.
	T []vsample

	// Temporary buffer.
	X []vsample

	// Current threshold
	tau float64

	// Size of sample & scale
	capacity int

	totalCount  int
	totalWeight float64
}

// Sample is an empty interface that represents a sample item.
// Sampling algorithms treat these as opaque, as their weight is
// passed in separately.
type Sample interface{}

type vsample struct {
	sample Sample
	weight float64
}

type largeHeap []vsample

// New returns a new Varopt sampler with given capacity (i.e.,
// reservoir size) and random number generator.
func New(capacity int, rnd *rand.Rand) *Varopt {
	return &Varopt{
		capacity: capacity,
		rnd:      rnd,
	}
}

// Add considers a new observation for the sample with given weight.
func (s *Varopt) Add(sample Sample, weight float64) {
	individual := vsample{
		sample: sample,
		weight: weight,
	}

	if weight <= 0 {
		panic(fmt.Sprint("Invalid weight <= 0: ", weight))
	}

	s.totalCount++
	s.totalWeight += weight

	if s.Size() < s.capacity {
		s.L.push(individual)
		return
	}

	// the X <- {} step from the paper is not done here,
	// but rather at the bottom of the function

	W := s.tau * float64(len(s.T))

	if weight > s.tau {
		s.L.push(individual)
	} else {
		s.X = append(s.X, individual)
		W += weight
	}

	for len(s.L) > 0 && W >= float64(len(s.T)+len(s.X)-1)*s.L[0].weight {
		h := s.L.pop()
		s.X = append(s.X, h)
		W += h.weight
	}

	s.tau = W / float64(len(s.T)+len(s.X)-1)
	r := s.uniform()
	d := 0

	for d < len(s.X) && r >= 0 {
		wxd := s.X[d].weight
		r -= (1 - wxd/s.tau)
		d++
	}
	if r < 0 {
		if d < len(s.X) {
			s.X[d], s.X[len(s.X)-1] = s.X[len(s.X)-1], s.X[d]
		}
		s.X = s.X[:len(s.X)-1]
	} else {
		ti := s.rnd.Intn(len(s.T))
		s.T[ti], s.T[len(s.T)-1] = s.T[len(s.T)-1], s.T[ti]
		s.T = s.T[:len(s.T)-1]
	}
	s.T = append(s.T, s.X...)
	s.X = s.X[:0]
}

func (s *Varopt) uniform() float64 {
	for {
		r := s.rnd.Float64()
		if r != 0.0 {
			return r
		}
	}
}

// Get() returns the i'th sample and its adjusted weight. To obtain
// the sample's original weight (i.e. what was passed to Add), use
// GetOriginalWeight(i).
func (s *Varopt) Get(i int) (Sample, float64) {
	if i < len(s.L) {
		return s.L[i].sample, s.L[i].weight
	}

	return s.T[i-len(s.L)].sample, s.tau
}

// GetOriginalWeight returns the original input weight of the sample
// item that was passed to Add().  This can be useful for computing a
// frequency from the adjusted sample weight.
func (s *Varopt) GetOriginalWeight(i int) float64 {
	if i < len(s.L) {
		return s.L[i].weight
	}

	return s.T[i-len(s.L)].weight
}

// Capacity returns the size of the reservoir.  This is the maximum
// size of the sample.
func (s *Varopt) Capacity() int {
	return s.capacity
}

// Size returns the current number of items in the sample.  If the
// reservoir is full, this returns Capacity().
func (s *Varopt) Size() int {
	return len(s.L) + len(s.T)
}

// TotalWeight returns the sum of weights that were passed to Add().
func (s *Varopt) TotalWeight() float64 {
	return s.totalWeight
}

// TotalCount returns the number of calls to Add().
func (s *Varopt) TotalCount() int {
	return s.totalCount
}

// Tau returns the current large-weight threshold.  Weights larger
// than Tau() carry their exact weight in the sample.  See the VarOpt
// paper for details.
func (s *Varopt) Tau() float64 {
	return s.tau
}

func (lp *largeHeap) push(v vsample) {
	l := append(*lp, v)
	n := len(l) - 1

	// This copies the body of heap.up().
	j := n
	for {
		i := (j - 1) / 2 // parent
		if i == j || l[j].weight >= l[i].weight {
			break
		}
		l[i], l[j] = l[j], l[i]
		j = i
	}

	*lp = l
}

func (lp *largeHeap) pop() vsample {
	l := *lp
	n := len(l) - 1
	result := l[0]
	l[0] = l[n]
	l = l[:n]

	// This copies the body of heap.down().
	i := 0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && l[j2].weight < l[j1].weight {
			j = j2 // = 2*i + 2  // right child
		}
		if l[j].weight >= l[i].weight {
			break
		}
		l[i], l[j] = l[j], l[i]
		i = j
	}

	*lp = l
	return result
}
