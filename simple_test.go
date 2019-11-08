// Copyright 2019, LightStep Inc.

package varopt_test

import (
	"math/rand"
	"testing"

	"github.com/lightstep/varopt"
	"github.com/stretchr/testify/require"
)

type iRec int

func TestSimple(t *testing.T) {
	const (
		popSize        = 1e6
		sampleProb     = 0.1
		sampleSize int = popSize * sampleProb
		epsilon        = 0.01
	)

	rnd := rand.New(rand.NewSource(17167))

	ss := varopt.NewSimple(sampleSize, rnd)

	psum := 0.
	for i := 0; i < popSize; i++ {
		ss.Add(iRec(i))
		psum += float64(i)
	}

	require.Equal(t, ss.Size(), sampleSize)

	ssum := 0.0
	for i := 0; i < sampleSize; i++ {
		ssum += float64(ss.Get(i).(iRec))
	}

	require.InEpsilon(t, ssum/float64(ss.Size()), psum/popSize, epsilon)
}
