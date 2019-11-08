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

func NewSimple(capacity int, rnd *rand.Rand) *Simple {
	return &Simple{
		capacity: capacity,
		rnd:      rnd,
	}
}

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

func (s *Simple) Get(i int) Sample {
	return s.buffer[i]
}

func (s *Simple) Size() int {
	return len(s.buffer)
}

func (s *Simple) Weight() float64 {
	return float64(s.observed) / float64(s.Size())
}

func (s *Simple) Prob() float64 {
	return 1 / s.Weight()
}

func (s *Simple) Observed() int {
	return s.observed
}
