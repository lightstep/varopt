// Stream sampling for variance-optimal estimation of subset sums
// Edith Cohen, Nick Duffield, Haim Kaplan, Carsten Lund, Mikkel Thorup
// 2008
// https://arxiv.org/pdf/0803.0473.pdf

package varopt

import (
	"container/heap"
	"fmt"
	"math/rand"
)

type Varopt struct {
	// Large-weight items
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

type vsample struct {
	sample Sample
	weight float64
}

type largeHeap []vsample

func NewVaropt(capacity int) *Varopt {
	v := InitVaropt(capacity)
	return &v
}

func InitVaropt(capacity int) Varopt {
	return Varopt{
		capacity: capacity,
	}
}

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
		heap.Push(&s.L, individual)
		return
	}

	// the X <- {} step from the paper is not done here,
	// but rather at the bottom of the function

	W := s.tau * float64(len(s.T))

	if weight > s.tau {
		heap.Push(&s.L, individual)
	} else {
		s.X = append(s.X, individual)
		W += weight
	}

	for len(s.L) > 0 && W >= float64(len(s.T)+len(s.X)-1)*s.L[0].weight {
		h := heap.Pop(&s.L).(vsample)
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
		ti := rand.Intn(len(s.T))
		s.T[ti], s.T[len(s.T)-1] = s.T[len(s.T)-1], s.T[ti]
		s.T = s.T[:len(s.T)-1]
	}
	s.T = append(s.T, s.X...)
	s.X = s.X[:0]
}

func (s *Varopt) uniform() float64 {
	for {
		r := rand.Float64()
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

func (s *Varopt) GetOriginalWeight(i int) float64 {
	if i < len(s.L) {
		return s.L[i].weight
	}

	return s.T[i-len(s.L)].weight
}

func (s *Varopt) Capacity() int {
	return s.capacity
}

func (s *Varopt) Size() int {
	return len(s.L) + len(s.T)
}

func (s *Varopt) TotalWeight() float64 {
	return s.totalWeight
}

func (s *Varopt) TotalCount() int {
	return s.totalCount
}

func (s *Varopt) Tau() float64 {
	return s.tau
}

func (b largeHeap) Len() int {
	return len(b)
}

func (b largeHeap) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b largeHeap) Less(i, j int) bool {
	return b[i].weight < b[j].weight
}

func (b *largeHeap) Push(x interface{}) {
	*b = append(*b, x.(vsample))
}

func (b *largeHeap) Pop() interface{} {
	old := *b
	n := len(old)
	x := old[n-1]
	*b = old[0 : n-1]
	return x
}
