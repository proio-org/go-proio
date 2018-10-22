package proio_test

import (
	"bytes"
	"fmt"

	"github.com/proio-org/go-proio"
	model "github.com/proio-org/go-proio-pb/model/example"
)

func Example_skip() {
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

	bytesReader := bytes.NewReader(buffer.Bytes())
	reader := proio.NewReader(bytesReader)

	reader.Skip(7)
	event, _ := reader.Next()
	fmt.Print(event)
	reader.SeekToStart()
	event, _ = reader.Next()
	fmt.Print(event)

	// Output:
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// pdg: 18
	//
	// ---------- TAG: Particle ----------
	// ID: 1
	// Entry type: proio.model.example.Particle
	// pdg: 11
}
