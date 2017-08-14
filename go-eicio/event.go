package eicio

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
)

type Event struct {
	Header  *EventHeader
	payload []byte
}

func NewEvent() *Event {
	return &Event{Header: &EventHeader{}}
}

func (evt *Event) String() string {
	buffer := &bytes.Buffer{}

	fmt.Fprint(buffer, "event header...\n", proto.MarshalTextString(evt.Header), "\n")
	for _, collHdr := range evt.Header.Collection {
		coll, err := evt.Get(collHdr.Name)
		if coll != nil && err == nil {
			fmt.Fprint(buffer, "collection name: ", collHdr.Name, "\n")
			fmt.Fprint(buffer, "type: ", collHdr.Type, "\n")
			fmt.Fprint(buffer, proto.MarshalTextString(coll), "\n")
		}
	}

	return string(buffer.Bytes())
}

func (evt *Event) Add(coll Message, name string) error {
	collHdr := &EventHeader_CollectionHeader{}
	collHdr.Name = name
	collHdr.Id = coll.GetId()
	collHdr.Type = strings.TrimPrefix(proto.MessageName(coll), "eicio.")

	collBuf, err := coll.Marshal()
	if err != nil {
		return err
	}
	collHdr.PayloadSize = uint32(len(collBuf))

	if evt.Header == nil {
		evt.Header = &EventHeader{}
	}
	evt.Header.Collection = append(evt.Header.Collection, collHdr)
	evt.payload = append(evt.payload, collBuf...)

	return nil
}

var ErrBlankColl = errors.New("collection not found or type is blank")

func (evt *Event) Get(name string) (Message, error) {
	offset := uint32(0)
	size := uint32(0)
	collType := ""
	for _, coll := range evt.Header.Collection {
		if coll.Name == name {
			collType = coll.Type
			size = coll.PayloadSize
			break
		}
		offset += coll.PayloadSize
	}
	if collType == "" {
		return nil, ErrBlankColl
	}

	msgType := proto.MessageType("eicio." + collType).Elem()
	coll := reflect.New(msgType).Interface().(Message)

	if err := coll.Unmarshal(evt.payload[offset : offset+size]); err != nil {
		return nil, err
	}
	return coll, nil
}

func (evt *Event) getPayload() []byte {
	return evt.payload
}

func (evt *Event) setPayload(payload []byte) {
	evt.payload = payload
}

type Message interface {
	proto.Message

	Marshal() ([]byte, error)
	Unmarshal([]byte) error

	GetId() uint32
}
