package proio

import (
	"bytes"
	"testing"

	"github.com/proio-org/go-proio-pb/model/eic"
	"github.com/proio-org/go-proio-pb/model/mc"
)

func TestWriteFD1(t *testing.T) {
	buffer := &bytes.Buffer{}

	writer := NewWriter(buffer)
	event := NewEvent()
	event.AddEntry("test", &eic.Particle{})
	event.AddEntry("test", &mc.Particle{})
	writer.Push(event)
	if len(writer.writtenFDs) != 2 {
		t.Errorf("Length of writtenFDs is %v", len(writer.writtenFDs))
	}
	writer.Close()

	reader := NewReader(buffer)
	event, _ = reader.Next()
	reader.Close()

	writer = NewWriter(buffer)
	event.AddEntry("test", &eic.Particle{})
	event.AddEntry("test", &mc.Particle{})
	writer.Push(event)
	if len(writer.writtenFDs) != 2 {
		t.Errorf("Length of writtenFDs is %v", len(writer.writtenFDs))
	}
	writer.Close()

	storedFDs := StoredFileDescriptorProtos()
	if len(storedFDs) == 0 {
		t.Errorf("Size of storedFDs is 0")
	}
}
