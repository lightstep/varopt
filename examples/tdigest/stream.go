package tdigest

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/lightstep/varopt"
)

type (
	QuantileFunc interface {
		Quantile(value float64) float64
	}

	Config struct {
		// quality determines the number of centroids used to
		// compute the digest.
		quality Quality

		// currentSize is the current sample size.
		currentSize int

		// completeSize is the complete sample size.
		completeSize int

		// windowSize is the number of points between
		// re-computing the digest.
		windowSize int
	}

	currentWindow struct {
		sample    *varopt.Varopt
		temporary []float64
	}

	completeWindow struct {
		sample    *varopt.Varopt
		temporary []float64
		freelist  []*float64
	}

	Stream struct {
		// Config stores parameters.
		Config

		// rnd is used by both both samplers.
		rnd *rand.Rand

		// Buffer and tinput used as inputs to T-digest.
		// Buffer is ordered by this code, tinput is sorted
		// while computing the digest. This is twice the
		// currentSize, since at each round we combine the
		// prior distribution with the current.  Note that
		// T-digest supports zero-weight inputs, so we always
		// pass entire slices, even when partially filled.
		buffer []Item
		tinput Input

		// The cycle count tells the number of T-digest
		// iterations, indicating which half of buffer and
		// weight to fill.
		cycle int

		// digest is the current estimated distribution.
		digest *TDigest

		// flushed prevents Add() after Quantile() is called.
		flushed bool

		// current stores a sample of the latest window of
		// data.
		current currentWindow

		// current stores a sample of the complete stream of data.
		complete completeWindow
	}
)

var _ QuantileFunc = &Stream{}

func NewConfig(quality Quality, currentSize, completeSize, windowSize int) Config {
	return Config{
		quality:      quality,
		currentSize:  currentSize,
		completeSize: completeSize,
		windowSize:   windowSize,
	}
}

func NewStream(config Config, rnd *rand.Rand) *Stream {
	bufsz := 2 * config.currentSize
	if config.completeSize > bufsz {
		bufsz = config.completeSize
	}

	stream := &Stream{
		Config: config,
		digest: New(config.quality),
		rnd:    rnd,
		buffer: make([]Item, bufsz),
		tinput: make([]*Item, bufsz),
	}

	for i := range stream.buffer {
		stream.tinput[i] = &stream.buffer[i]
	}

	stream.current = stream.newCurrentWindow(config.currentSize)
	stream.complete = stream.newCompleteWindow(config.completeSize)
	return stream
}

func (s *Stream) newCurrentWindow(sampleSize int) currentWindow {
	return currentWindow{
		sample:    varopt.New(sampleSize, s.rnd),
		temporary: make([]float64, 0, s.windowSize),
	}
}

func (v *currentWindow) add(value, weight float64) error {
	v.temporary = append(v.temporary, value)
	_, err := v.sample.Add(&v.temporary[len(v.temporary)-1], weight)
	return err
}

func (v *currentWindow) full() bool {
	return len(v.temporary) == cap(v.temporary)
}

func (v *currentWindow) size() int {
	return v.sample.Size()
}

func (v *currentWindow) get(i int) (interface{}, float64) {
	value, weight := v.sample.Get(i)
	freq := weight / v.sample.GetOriginalWeight(i)

	if math.IsNaN(freq) {
		panic(fmt.Sprintln("NaN here", value, weight))
	}

	return value, freq
}

func (v *currentWindow) restart() {
	v.temporary = v.temporary[:0]
	v.sample.Reset()
}

func (s *Stream) newCompleteWindow(sampleSize int) completeWindow {
	cw := completeWindow{
		sample:    varopt.New(sampleSize, s.rnd),
		temporary: make([]float64, sampleSize+1),
		freelist:  make([]*float64, sampleSize+1),
	}
	for i := range cw.temporary {
		cw.freelist[i] = &cw.temporary[i]
	}
	return cw
}

func (v *completeWindow) add(value, weight float64) error {
	lastdata := len(v.freelist) - 1
	data := v.freelist[lastdata]
	v.freelist = v.freelist[:lastdata]

	*data = value
	eject, err := v.sample.Add(data, weight)

	if err != nil {
		v.freelist = append(v.freelist, data)
	} else if eject != nil {
		v.freelist = append(v.freelist, eject.(*float64))
	}
	return err
}

func (v *completeWindow) size() int {
	return v.sample.Size()
}

func (v *completeWindow) get(i int) (interface{}, float64) {
	value, weight := v.sample.Get(i)
	freq := weight / v.sample.GetOriginalWeight(i)
	return value, freq
}

func (s *Stream) Add(x float64) error {
	var weight float64

	if s.cycle == 0 {
		weight = 1
	} else {
		prob := s.digest.Probability(x)
		weight = 1 / prob
	}

	if err := s.current.add(x, weight); err != nil {
		return err
	}

	if !s.current.full() {
		return nil
	}

	return s.recompute()
}

func (s *Stream) recompute() error {
	// Recompute the T-Digest from the new weighted sample
	// combined with the prior data.  This computes a MAP
	// estimate.
	offset := s.currentSize * (s.cycle % 2)

	for i := 0; i < s.current.size(); i++ {
		valueI, freq := s.current.get(i)
		value := *(valueI.(*float64))

		//fmt.Println("New W", value, freq)

		s.buffer[offset+i].Value = value
		s.buffer[offset+i].Weight = freq

		s.complete.add(value, freq)
	}

	// Fill in zero-weight values in case the sample size was
	// smaller than capacity at the end of the stream.
	for i := s.current.size(); i < s.currentSize; i++ {
		s.buffer[offset+i] = Item{}
	}

	if err := s.digest.Compute(s.tinput[0 : 2*s.currentSize]); err != nil {
		return err
	}

	s.current.restart()
	s.cycle++
	return nil
}

func (s *Stream) flush() error {
	if s.flushed {
		// Note: this could be fixed if needed.
		return fmt.Errorf("Do not Add() after calling Quantile()")
	}
	s.flushed = true

	for i := 0; i < s.current.size(); i++ {
		valueI, freq := s.current.get(i)
		value := *(valueI.(*float64))

		s.complete.add(value, freq)
	}

	sumWeight := 0.0
	for i := 0; i < s.complete.size(); i++ {
		valueI, freq := s.complete.get(i)
		value := *(valueI.(*float64))

		s.buffer[i].Value = value
		s.buffer[i].Weight = freq
		s.tinput[i] = &s.buffer[i]
		sumWeight += freq
	}

	s.tinput = s.tinput[0:s.complete.size()]
	sort.Sort(&s.tinput)

	sum := 0.0
	for _, t := range s.tinput {
		sum += t.Weight
		t.Weight = sum / sumWeight
	}

	return nil
}

// Quantile returns the estimated value for a given quantile.
//
// Note: TDigest can implement QuantileFunc itself, but in this
// implementation we use the stream sample directly. See section 2.9
// of the T-digest paper for recommendations about interpolating the
// CDF of a T-digest.
func (s *Stream) Quantile(quantile float64) float64 {
	if !s.flushed {
		s.flush()
	}

	idx := sort.Search(len(s.tinput), func(i int) bool {
		return quantile <= s.tinput[i].Weight
	})

	if idx == len(s.tinput) {
		return s.tinput[len(s.tinput)-1].Value
	}
	return s.tinput[idx].Value
}
