// Copyright 2019, LightStep Inc.

package varopt_test

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/lightstep/varopt"
)

type packet struct {
	size     int
	color    string
	protocol string
}

func ExampleNew() {
	const totalPackets = 1e6
	const sampleRatio = 0.01

	colors := []string{"red", "green", "blue"}
	protocols := []string{"http", "tcp", "udp"}

	sizeByColor := map[string]int{}
	sizeByProtocol := map[string]int{}
	trueTotalWeight := 0.0

	rnd := rand.New(rand.NewSource(32491))
	sampler := varopt.New(totalPackets*sampleRatio, rnd)

	for i := 0; i < totalPackets; i++ {
		packet := packet{
			size:     1 + rnd.Intn(100000),
			color:    colors[rnd.Intn(len(colors))],
			protocol: protocols[rnd.Intn(len(protocols))],
		}

		sizeByColor[packet.color] += packet.size
		sizeByProtocol[packet.protocol] += packet.size
		trueTotalWeight += float64(packet.size)

		sampler.Add(packet, float64(packet.size))
	}

	estSizeByColor := map[string]float64{}
	estSizeByProtocol := map[string]float64{}
	estTotalWeight := 0.0

	for i := 0; i < sampler.Size(); i++ {
		sample, weight := sampler.Get(i)
		packet := sample.(packet)
		estSizeByColor[packet.color] += weight
		estSizeByProtocol[packet.protocol] += weight
		estTotalWeight += weight
	}

	// Compute mean average percentage error for colors
	colorMape := 0.0
	for _, c := range colors {
		colorMape += math.Abs(float64(sizeByColor[c])-estSizeByColor[c]) / float64(sizeByColor[c])
	}
	colorMape /= float64(len(colors))

	// Compute mean average percentage error for protocols
	protocolMape := 0.0
	for _, p := range protocols {
		protocolMape += math.Abs(float64(sizeByProtocol[p])-estSizeByProtocol[p]) / float64(sizeByProtocol[p])
	}
	protocolMape /= float64(len(protocols))

	// Compute total sum error percentage
	fmt.Printf("Total sum error %.2g%%\n", 100*math.Abs(estTotalWeight-trueTotalWeight)/trueTotalWeight)
	fmt.Printf("Color mean absolute percentage error %.2f%%\n", 100*colorMape)
	fmt.Printf("Protocol mean absolute percentage error %.2f%%\n", 100*protocolMape)

	// Output:
	// Total sum error 2.4e-11%
	// Color mean absolute percentage error 0.73%
	// Protocol mean absolute percentage error 1.62%
}
