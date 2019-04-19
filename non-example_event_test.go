package proio

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	protobuf "github.com/golang/protobuf/proto"
	"github.com/proio-org/go-proio-pb/model/eic"
	"github.com/proio-org/go-proio-pb/model/example"
	prolcio "github.com/proio-org/go-proio-pb/model/lcio"
)

func TestStrip1(t *testing.T) {
	event := NewEvent()
	event.AddEntry(
		"Particle",
		&eic.Particle{},
	)
	event.AddEntry(
		"Particle",
		&eic.Particle{},
	)
	event.AddEntry(
		"SimHit",
		&eic.SimHit{},
	)

	for _, ID := range event.TaggedEntries("Particle") {
		event.RemoveEntry(ID)
	}

	nEntries := len(event.AllEntries())
	if nEntries != 1 {
		t.Errorf("There should be one entry, but len(event.AllEntries()) = %v", nEntries)
	}
}

func TestTagUntag1(t *testing.T) {
	event := NewEvent()
	id0 := event.AddEntry(
		"MCParticles",
		&prolcio.MCParticle{},
	)
	id1 := event.AddEntry(
		"MCParticles",
		&prolcio.MCParticle{},
	)
	event.UntagEntry(id0, "MCParticles")

	mcIDs := event.TaggedEntries("MCParticles")
	if len(mcIDs) != 1 {
		t.Errorf("%v IDs instead of 1", len(mcIDs))
	}
	if mcIDs[0] != id1 {
		t.Errorf("got ID %v instead of %v", mcIDs[0], id1)
	}
}

func TestTagUntag2(t *testing.T) {
	event := NewEvent()
	event.AddEntry(
		"MCParticles",
		&prolcio.MCParticle{},
	)
	event.AddEntry(
		"MCParticles",
		&prolcio.MCParticle{},
	)
	event.DeleteTag("MCParticles")

	mcIDs := event.TaggedEntries("MCParticles")
	if len(mcIDs) != 0 {
		t.Errorf("%v IDs instead of 0", len(mcIDs))
	}
}

func TestTagUntag3(t *testing.T) {
	event := NewEvent()
	id0 := event.AddEntry(
		"Particle",
		&prolcio.MCParticle{},
	)
	event.AddEntry(
		"Particle",
		&prolcio.MCParticle{},
	)
	event.UntagEntry(id0, "MCParticles")

	ids := event.TaggedEntries("Particle")
	if len(ids) != 2 {
		t.Errorf("%v IDs instead of 2", len(ids))
	}
}

func TestRevTagLookup(t *testing.T) {
	event := NewEvent()
	id := event.AddEntry(
		"MCParticles",
		&prolcio.MCParticle{},
	)
	event.TagEntry(id, "Simulated", "Particles")

	tags := event.EntryTags(id)
	if tags[0] != "MCParticles" {
		t.Errorf("First tag is %v instead of MCParticles", tags[0])
	}
	if tags[1] != "Particles" {
		t.Errorf("Second tag is %v instead of Particles", tags[1])
	}
	if tags[2] != "Simulated" {
		t.Errorf("Third tag is %v instead of Simulated", tags[2])
	}
}

func TestGetTags(t *testing.T) {
	event := NewEvent()
	id := event.AddEntry(
		"MCParticles",
		&prolcio.MCParticle{},
	)
	event.TagEntry(id, "Simulated", "Particles")

	tags := event.Tags()
	if tags[0] != "MCParticles" {
		t.Errorf("First tag is %v instead of MCParticles", tags[0])
	}
	if tags[1] != "Particles" {
		t.Errorf("Second tag is %v instead of Particles", tags[1])
	}
	if tags[2] != "Simulated" {
		t.Errorf("Third tag is %v instead of Simulated", tags[2])
	}
}

func TestDeleteTag(t *testing.T) {
	event := NewEvent()
	id := event.AddEntry(
		"MCParticles",
		&prolcio.MCParticle{},
	)
	event.TagEntry(id, "Simulated", "Particles")

	event.DeleteTag("Particles")

	tags := event.Tags()
	if tags[0] != "MCParticles" {
		t.Errorf("First tag is %v instead of MCParticles", tags[0])
	}
	if tags[1] != "Simulated" {
		t.Errorf("Second tag is %v instead of Simulated", tags[2])
	}
}

func TestDirtyTag(t *testing.T) {
	event := NewEvent()
	id0 := event.AddEntry(
		"Particle",
		&eic.Particle{},
	)
	id1 := event.AddEntry(
		"Particle",
		&eic.Particle{},
	)
	event.RemoveEntry(id0)

	ids := event.TaggedEntries("Particle")
	if len(ids) != 1 {
		t.Errorf("%v IDs instead of 1", len(ids))
	}
	if ids[0] != id1 {
		t.Errorf("got ID %v instead of %v", ids[0], id1)
	}
}

func TestNoSuchEntry(t *testing.T) {
	event := NewEvent()
	entry := event.GetEntry(0)
	if entry != nil {
		t.Error("Entry is not nil")
	}
}

type unknownMsg struct {
}

func (*unknownMsg) Reset()         {}
func (*unknownMsg) String() string { return "" }
func (*unknownMsg) ProtoMessage()  {}

func TestUnknownType(t *testing.T) {
	event := NewEvent()
	id := event.AddEntry("unknown", &unknownMsg{})

	writer := NewWriter(&bytes.Buffer{})
	writer.Push(event)

	entry := event.GetEntry(id)
	if entry != nil {
		t.Error("Event returns entry for unknown type")
	}
}

type nonSelfSerializingMsg struct {
	X float32 `protobuf:"fixed32,1,opt,name=x,proto3" json:"x,omitempty"`
	Y float32 `protobuf:"fixed32,2,opt,name=y,proto3" json:"y,omitempty"`
	Z float32 `protobuf:"fixed32,3,opt,name=z,proto3" json:"z,omitempty"`
}

func (*nonSelfSerializingMsg) Reset()         {}
func (*nonSelfSerializingMsg) String() string { return "" }
func (*nonSelfSerializingMsg) ProtoMessage()  {}

func init() {
	protobuf.RegisterType((*nonSelfSerializingMsg)(nil), "nonSelfSerializingMsg")
}

func TestNonSelfSerializingMsg(t *testing.T) {
	event := NewEvent()
	id := event.AddEntry("nonSelfSerializingMsg", &nonSelfSerializingMsg{})

	writer := NewWriter(&bytes.Buffer{})
	writer.Push(event)

	entry := event.GetEntry(id)
	if entry == nil {
		t.Error("Unable to deserialize message")
	}
}

type halfSelfSerializingMsg1 struct {
	X float32 `protobuf:"fixed32,1,opt,name=x,proto3" json:"x,omitempty"`
	Y float32 `protobuf:"fixed32,2,opt,name=y,proto3" json:"y,omitempty"`
	Z float32 `protobuf:"fixed32,3,opt,name=z,proto3" json:"z,omitempty"`
}

func (*halfSelfSerializingMsg1) Reset()                   {}
func (*halfSelfSerializingMsg1) String() string           { return "" }
func (*halfSelfSerializingMsg1) ProtoMessage()            {}
func (*halfSelfSerializingMsg1) Marshal() ([]byte, error) { return []byte{0x7}, nil }

func init() {
	protobuf.RegisterType((*halfSelfSerializingMsg1)(nil), "halfSelfSerializingMsg1")
}

func TestHalfSelfSerializingMsg1(t *testing.T) {
	event := NewEvent()
	id := event.AddEntry("halfSelfSerializingMsg1", &halfSelfSerializingMsg1{})

	writer := NewWriter(&bytes.Buffer{})
	writer.Push(event)

	entry := event.GetEntry(id)
	if entry != nil {
		t.Error("Broken message returns non-nil value")
	}

	eventString := `---------- TAG: halfSelfSerializingMsg1 ----------
ID: 1
failure to unmarshal entry 1 with type halfSelfSerializingMsg1
`
	if event.String() != eventString {
		t.Errorf("Event string is \n%v\ninstead of\n%v", event.String(), eventString)
	}
}

type halfSelfSerializingMsg2 struct {
	X float32 `protobuf:"fixed32,1,opt,name=x,proto3" json:"x,omitempty"`
	Y float32 `protobuf:"fixed32,2,opt,name=y,proto3" json:"y,omitempty"`
	Z float32 `protobuf:"fixed32,3,opt,name=z,proto3" json:"z,omitempty"`
}

func (*halfSelfSerializingMsg2) Reset()                 {}
func (*halfSelfSerializingMsg2) String() string         { return "" }
func (*halfSelfSerializingMsg2) ProtoMessage()          {}
func (*halfSelfSerializingMsg2) Unmarshal([]byte) error { return errors.New("bad") }

func init() {
	protobuf.RegisterType((*halfSelfSerializingMsg2)(nil), "halfSelfSerializingMsg2")
}

func TestHalfSelfSerializingMsg2(t *testing.T) {
	event := NewEvent()
	id := event.AddEntry("halfSelfSerializingMsg2", &halfSelfSerializingMsg2{})

	writer := NewWriter(&bytes.Buffer{})
	writer.Push(event)

	entry := event.GetEntry(id)
	if entry != nil {
		t.Error("Broken message returns non-nil value")
	}

	eventString := `---------- TAG: halfSelfSerializingMsg2 ----------
ID: 1
failure to unmarshal entry 1 with type halfSelfSerializingMsg2
`
	if event.String() != eventString {
		t.Errorf("Event string is \n%v\ninstead of\n%v", event.String(), eventString)
	}
}

func TestCopyEvent(t *testing.T) {
	event := NewEvent()
	event.AddEntry(
		"Test",
		&example.Particle{},
	)
	event.Metadata["md1"] = []byte{0x0}

	newEvent := CopyEvent(event)

	if event.String() != newEvent.String() {
		t.Errorf("Copied event does not have equivalent data")
	}
	if !reflect.DeepEqual(event.Metadata, newEvent.Metadata) {
		t.Errorf("Copied event does not have equivalent metadata")
	}

	event.AddEntry(
		"Test",
		&example.Particle{},
	)
	event.FlushCache()
	event.Metadata["md1"] = []byte{0x1}

	if event.String() == newEvent.String() {
		t.Errorf("New event data are not independent")
	}
	if reflect.DeepEqual(event.Metadata, newEvent.Metadata) {
		t.Errorf("New event metadata are not independent")
	}
}

func TestAddSerializedEntry(t *testing.T) {
	desc, _ := (&example.Particle{}).Descriptor()

	event := NewEvent()
	id, err := event.AddSerializedEntry(
		"Test",
		[]byte{},
		"proio.model.example.Particle",
		desc,
	)
	if err != nil {
		t.Errorf(err.Error())
	}
	if event.GetEntry(id) == nil {
		t.Errorf("Entry failed to deserialize")
	}

	id, err = event.AddSerializedEntry(
		"Test",
		[]byte{},
		"proio.model.example.NotReal",
		[]byte("garbage"),
	)
	if err == nil {
		t.Errorf("bad descriptor data not caught")
	}
	if event.GetEntry(id) != nil {
		t.Errorf("fake message type somehow deserialized?")
	}
}
