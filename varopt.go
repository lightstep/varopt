// Copyright 2019, LightStep Inc.

package varopt

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/lightstep/varopt/internal"
)

// Varopt implements the algorithm from Stream sampling for
// variance-optimal estimation of subset sums Edith Cohen, Nick
// Duffield, Haim Kaplan, Carsten Lund, Mikkel Thorup 2008
//
// https://arxiv.org/pdf/0803.0473.pdf
type Varopt[T any] struct {
	// Random number generator
	rnd *rand.Rand

	// Large-weight items stored in a min-heap.
	L internal.SampleHeap[T]

	// Light-weight items.
	T []internal.Vsample[T]

	// Temporary buffer.
	X []internal.Vsample[T]

	// Current threshold
	tau float64

	// Size of sample & scale
	capacity int

	totalCount  int
	totalWeight float64
}

var ErrInvalidWeight = fmt.Errorf("Negative, Zero, Inf or NaN weight")

// New returns a new Varopt sampler with given capacity (i.e.,
// reservoir size) and random number generator.
func New[T any](capacity int, rnd *rand.Rand) *Varopt[T] {
	v := &Varopt[T]{}
	v.Init(capacity, rnd)
	return v
}

func (v *Varopt[T]) Init(capacity int, rnd *rand.Rand) {
	*v = Varopt[T]{
		capacity: capacity,
		rnd:      rnd,
		L:        make(internal.SampleHeap[T], 0, capacity),
		T:        make(internal.SampleHeap[T], 0, capacity),
	}
}

// Reset returns the sampler to its initial state, maintaining its
// capacity and random number source.
func (s *Varopt[T]) Reset() {
	s.L = s.L[:0]
	s.T = s.T[:0]
	s.X = s.X[:0]
	s.tau = 0
	s.totalCount = 0
	s.totalWeight = 0
}

// Add considers a new observation for the sample with given weight.
// If there is an item ejected from the sample as a result, the item
// is returned to allow re-use of memory.
//
// An error will be returned if the weight is either negative or NaN.
func (s *Varopt[T]) Add(item T, weight float64) (T, error) {
	var zero T
	individual := internal.Vsample[T]{
		Sample: item,
		Weight: weight,
	}

	if weight <= 0 || math.IsNaN(weight) || math.IsInf(weight, 1) {
		return zero, ErrInvalidWeight
	}

	s.totalCount++
	s.totalWeight += weight

	if s.Size() < s.capacity {
		s.L.Push(individual)
		return zero, nil
	}

	// the X <- {} step from the paper is not done here,
	// but rather at the bottom of the function

	W := s.tau * float64(len(s.T))

	if weight > s.tau {
		s.L.Push(individual)
	} else {
		s.X = append(s.X, individual)
		W += weight
	}

	for len(s.L) > 0 && W >= float64(len(s.T)+len(s.X)-1)*s.L[0].Weight {
		h := s.L.Pop()
		s.X = append(s.X, h)
		W += h.Weight
	}

	s.tau = W / float64(len(s.T)+len(s.X)-1)
	r := s.uniform()
	d := 0

	for d < len(s.X) && r >= 0 {
		wxd := s.X[d].Weight
		r -= (1 - wxd/s.tau)
		d++
	}
	var eject T
	if r < 0 {
		if d < len(s.X) {
			s.X[d], s.X[len(s.X)-1] = s.X[len(s.X)-1], s.X[d]
		}
		eject = s.X[len(s.X)-1].Sample
		s.X = s.X[:len(s.X)-1]
	} else {
		ti := s.rnd.Intn(len(s.T))
		s.T[ti], s.T[len(s.T)-1] = s.T[len(s.T)-1], s.T[ti]
		eject = s.T[len(s.T)-1].Sample
		s.T = s.T[:len(s.T)-1]
	}
	s.T = append(s.T, s.X...)
	s.X = s.X[:0]
	return eject, nil
}

func (s *Varopt[T]) uniform() float64 {
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
func (s *Varopt[T]) Get(i int) (T, float64) {
	if i < len(s.L) {
		return s.L[i].Sample, s.L[i].Weight
	}

	return s.T[i-len(s.L)].Sample, s.tau
}

// GetOriginalWeight returns the original input weight of the sample
// item that was passed to Add().  This can be useful for computing a
// frequency from the adjusted sample weight.
func (s *Varopt[T]) GetOriginalWeight(i int) float64 {
	if i < len(s.L) {
		return s.L[i].Weight
	}

	return s.T[i-len(s.L)].Weight
}

// Capacity returns the size of the reservoir.  This is the maximum
// size of the sample.
func (s *Varopt[T]) Capacity() int {
	return s.capacity
}

// Size returns the current number of items in the sample.  If the
// reservoir is full, this returns Capacity().
func (s *Varopt[T]) Size() int {
	return len(s.L) + len(s.T)
}

// TotalWeight returns the sum of weights that were passed to Add().
func (s *Varopt[T]) TotalWeight() float64 {
	return s.totalWeight
}

// TotalCount returns the number of calls to Add().
func (s *Varopt[T]) TotalCount() int {
	return s.totalCount
}

// Tau returns the current large-weight threshold.  Weights larger
// than Tau() carry their exact weight in the sample.  See the VarOpt
// paper for details.
func (s *Varopt[T]) Tau() float64 {
	return s.tau
}
