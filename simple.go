// Copyright 2019, LightStep Inc.

package varopt

import (
	"math/rand"
)

// Simple implements unweighted reservoir sampling using Algorithm R
// from "Random sampling with a reservoir" by Jeffrey Vitter (1985)
// https://en.wikipedia.org/wiki/Reservoir_sampling#Algorithm_R
type Simple struct {
	capacity int
	observed int
	buffer   []Sample
	rnd      *rand.Rand
}

// NewSimple returns a simple reservoir sampler with given capacity
// (i.e., reservoir size) and random number generator.
func NewSimple(capacity int, rnd *rand.Rand) *Simple {
	return &Simple{
		capacity: capacity,
		rnd:      rnd,
	}
}

// Add considers a new observation for the sample.  Items have unit
// weight.
func (s *Simple) Add(span Sample) {
	s.observed++

	if s.buffer == nil {
		s.buffer = make([]Sample, 0, s.capacity)
	}

	if len(s.buffer) < s.capacity {
		s.buffer = append(s.buffer, span)
		return
	}

	// Give this a capacity/observed chance of replacing an existing entry.
	index := s.rnd.Intn(s.observed)
	if index < s.capacity {
		s.buffer[index] = span
	}
}

// Get returns the i'th selected item from the sample.
func (s *Simple) Get(i int) Sample {
	return s.buffer[i]
}

// Get returns the number of items in the sample.  If the reservoir is
// full, Size() equals Capacity().
func (s *Simple) Size() int {
	return len(s.buffer)
}

// Weight returns the adjusted weight of each item in the sample.
func (s *Simple) Weight() float64 {
	return float64(s.observed) / float64(s.Size())
}

// Count returns the number of items that were observed.
func (s *Simple) Count() int {
	return s.observed
}
