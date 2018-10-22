package proio_test

import (
	"bytes"
	"fmt"

	"github.com/proio-org/go-proio"
	model "github.com/proio-org/go-proio-pb/model/example"
)

func Example_scan() {
	buffer := &bytes.Buffer{}
	writer := proio.NewWriter(buffer)

	for i := 0; i < 8; i++ {
		event := proio.NewEvent()
		p := &model.Particle{
			Pdg: int32(11 + i),
		}
		event.AddEntry("Particle", p)
		writer.Push(event)
	}
	writer.Flush()

	reader := proio.NewReader(buffer)

	for event := range reader.ScanEvents() {
		fmt.Print(event)
	}

	// Output:
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// pdg: 11
	//
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// pdg: 12
	//
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// pdg: 13
	//
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// pdg: 14
	//
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// pdg: 15
	//
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// pdg: 16
	//
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// pdg: 17
	//
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// pdg: 18
}
