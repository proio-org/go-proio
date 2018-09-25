package proio

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/proio-org/go-proio-pb/model/eic"
	prolcio "github.com/proio-org/go-proio-pb/model/lcio"
)

func doWrite(writer *Writer, b *testing.B, n int) {
	if b.N < 1000 {
		b.N = 1000
	}

	event := NewEvent()

	for i := 0; i < n-1; i++ {
		event.AddEntry("SimCaloHits", &prolcio.SimCalorimeterHit{
			Energy: rand.Float32(),
			Pos: []float32{
				rand.Float32(),
				rand.Float32(),
				rand.Float32(),
			},
		})
	}

	event.AddEntry("SimTrackHits", &prolcio.SimTrackerHit{
		EDep: rand.Float32(),
		Pos: []float64{
			rand.Float64(),
			rand.Float64(),
			rand.Float64(),
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writer.Push(event)
	}
	writer.Flush()
}

func doRead(reader *Reader, b *testing.B) {
	b.ResetTimer()
	for event := range reader.ScanEvents() {
		trackHitID := event.TaggedEntries("SimTrackHits")[0]
		_ = event.GetEntry(trackHitID)
	}
}

func BenchmarkWrite1000EntriesUncomp(b *testing.B) {
	buffer := &bytes.Buffer{}
	writer := NewWriter(buffer)
	writer.SetCompression(UNCOMPRESSED)

	doWrite(writer, b, 1000)
}

func BenchmarkWrite10000EntriesUncomp(b *testing.B) {
	buffer := &bytes.Buffer{}
	writer := NewWriter(buffer)
	writer.SetCompression(UNCOMPRESSED)

	doWrite(writer, b, 10000)
}

func BenchmarkWrite1000EntriesLZ4(b *testing.B) {
	buffer := &bytes.Buffer{}
	writer := NewWriter(buffer)
	writer.SetCompression(LZ4)

	doWrite(writer, b, 1000)
}

func BenchmarkWrite1000EntriesGZIP(b *testing.B) {
	buffer := &bytes.Buffer{}
	writer := NewWriter(buffer)
	writer.SetCompression(GZIP)

	doWrite(writer, b, 1000)
}

func BenchmarkWrite1000EntriesLZMA(b *testing.B) {
	buffer := &bytes.Buffer{}
	writer := NewWriter(buffer)
	writer.SetCompression(LZMA)

	doWrite(writer, b, 1000)
}

func BenchmarkReadWith1000EntriesUncomp(b *testing.B) {
	buffer := &bytes.Buffer{}
	writer := NewWriter(buffer)
	writer.SetCompression(UNCOMPRESSED)
	doWrite(writer, b, 1000)

	reader := NewReader(buffer)
	doRead(reader, b)
}

func BenchmarkReadWith10000EntriesUncomp(b *testing.B) {
	buffer := &bytes.Buffer{}
	writer := NewWriter(buffer)
	writer.SetCompression(UNCOMPRESSED)
	doWrite(writer, b, 10000)

	reader := NewReader(buffer)
	doRead(reader, b)
}

func BenchmarkReadWith1000EntriesLZ4(b *testing.B) {
	buffer := &bytes.Buffer{}
	writer := NewWriter(buffer)
	writer.SetCompression(LZ4)
	doWrite(writer, b, 1000)

	reader := NewReader(buffer)
	doRead(reader, b)
}

func BenchmarkReadWith1000EntriesGZIP(b *testing.B) {
	buffer := &bytes.Buffer{}
	writer := NewWriter(buffer)
	writer.SetCompression(GZIP)
	doWrite(writer, b, 1000)

	reader := NewReader(buffer)
	doRead(reader, b)
}

func BenchmarkReadWith1000EntriesLZMA(b *testing.B) {
	buffer := &bytes.Buffer{}
	writer := NewWriter(buffer)
	writer.SetCompression(LZMA)
	doWrite(writer, b, 1000)

	reader := NewReader(buffer)
	doRead(reader, b)
}

func BenchmarkAddRemove100Entries(b *testing.B) {
	addRemoveNEntries(b, 100)
}

func BenchmarkAddRemove1000Entries(b *testing.B) {
	addRemoveNEntries(b, 1000)
}

func BenchmarkAddRemove10000Entries(b *testing.B) {
	addRemoveNEntries(b, 10000)
}

func BenchmarkAddRemove100000Entries(b *testing.B) {
	addRemoveNEntries(b, 100000)
}

func addRemoveNEntries(b *testing.B, n int) {
	for i := 0; i < b.N; i++ {
		event := NewEvent()
		for i := 0; i < n; i++ {
			event.AddEntry("Particle", &eic.Particle{})
		}
		for i := 0; i < n; i++ {
			event.RemoveEntry(uint64(i + 1))
		}
	}
}
