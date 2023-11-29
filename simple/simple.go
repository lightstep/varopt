// Copyright 2019, LightStep Inc.

package simple

import (
	"math/rand"
)

// Simple implements unweighted reservoir sampling using Algorithm R
// from "Random sampling with a reservoir" by Jeffrey Vitter (1985)
// https://en.wikipedia.org/wiki/Reservoir_sampling#Algorithm_R
type Simple[T any] struct {
	capacity int
	observed int
	buffer   []T
	rnd      *rand.Rand
}

// New returns a simple reservoir sampler with given capacity
// (i.e., reservoir size) and random number generator.
func New[T any](capacity int, rnd *rand.Rand) *Simple[T] {
	s := &Simple[T]{}
	s.Init(capacity, rnd)
	return s
}

func (s *Simple[T]) Init(capacity int, rnd *rand.Rand) {
	*s = Simple[T]{
		capacity: capacity,
		buffer:   make([]T, 0, s.capacity),
		rnd:      rnd,
	}
}

// Add considers a new observation for the sample.  Items have unit
// weight.
func (s *Simple[T]) Add(item T) {
	s.observed++

	if len(s.buffer) < s.capacity {
		s.buffer = append(s.buffer, item)
		return
	}

	// Give this a capacity/observed chance of replacing an existing entry.
	index := s.rnd.Intn(s.observed)
	if index < s.capacity {
		s.buffer[index] = item
	}
}

// Get returns the i'th selected item from the sample.
func (s *Simple[T]) Get(i int) T {
	return s.buffer[i]
}

// Size returns the number of items in the sample.  If the reservoir is
// full, Size() equals Capacity().
func (s *Simple[T]) Size() int {
	return len(s.buffer)
}

// Count returns the number of items that were observed.
func (s *Simple[T]) Count() int {
	return s.observed
}
