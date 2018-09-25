package proio

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"reflect"
	"sync"

	protobuf "github.com/golang/protobuf/proto"
	"github.com/pierrec/lz4"
	proto "github.com/proio-org/go-proio-pb"
	"github.com/smira/lzma"
)

type Compression int

const (
	UNCOMPRESSED Compression = iota
	GZIP
	LZ4
	LZMA
)

// Writer serves to write Events into a stream in the proio format.  The Writer
// is not inherently thread safe, but it conveniently embeds sync.Mutex so that
// it can be locked and unlocked.
type Writer struct {
	streamWriter io.Writer
	bucket       *bytes.Buffer
	bucketHeader proto.BucketHeader
	metadata     map[string][]byte
	writtenFDs   map[protobuf.Message]bool

	deferredUntilClose []func() error

	sync.Mutex
}

// Create makes a new file specified by filename, overwriting any existing
// file, and returns a Writer for the file.  Either NewWriter or Create must be
// used to construct a Writer.
func Create(filename string) (*Writer, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	writer := NewWriter(file)
	writer.deferUntilClose(file.Close)

	return writer, nil
}

// Flush flushes any of the Writer's bucket contents.
func (wrt *Writer) Flush() error {
	if wrt.bucket.Len() > 0 {
		err := wrt.writeBucket()
		if err != nil {
			return err
		}
	}
	return nil
}

// Close calls Flush and closes any file that was created by the library.
// Close does not close io.Writers passed directly to NewWriter.
func (wrt *Writer) Close() error {
	for _, thisFunc := range wrt.deferredUntilClose {
		if err := thisFunc(); err != nil {
			return err
		}
	}
	return nil
}

// NewWriter takes an io.Writer and wraps it in a new proio Writer.  Either
// NewWriter or Create must be used to construct a Writer.
func NewWriter(streamWriter io.Writer) *Writer {
	writer := &Writer{
		streamWriter: streamWriter,
		bucket:       &bytes.Buffer{},
		metadata:     make(map[string][]byte),
		writtenFDs:   make(map[protobuf.Message]bool),
	}

	writer.SetCompression(GZIP)
	writer.deferUntilClose(writer.Flush)

	return writer
}

// Set compression type, for example to GZIP or UNCOMPRESSED.  This can be
// called even after writing some events.
func (wrt *Writer) SetCompression(comp Compression) error {
	switch comp {
	case LZMA:
		wrt.bucketHeader.Compression = proto.BucketHeader_LZMA
	case GZIP:
		wrt.bucketHeader.Compression = proto.BucketHeader_GZIP
	case LZ4:
		wrt.bucketHeader.Compression = proto.BucketHeader_LZ4
	case UNCOMPRESSED:
		wrt.bucketHeader.Compression = proto.BucketHeader_NONE
	default:
		return errors.New("invalid compression type")
	}

	return nil
}

type getDependencyer interface {
	GetDependency() []string
}

// Serialize the given Event.  Once this is performed, changes to the Event in
// memory are not reflected in the output stream.
func (wrt *Writer) Push(event *Event) error {
	for key, value := range event.Metadata {
		if !reflect.DeepEqual(wrt.metadata[key], value) {
			wrt.PushMetadata(key, value)
			wrt.metadata[key] = value
		}
	}

	event.FlushCache()
	protoBuf, err := event.proto.Marshal()
	if err != nil {
		return err
	}

	protoSizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(protoSizeBuf, uint32(len(protoBuf)))

	// add new protobuf FileDescriptorProtos to the stream that are required to
	// describe the event data
	newFDs := make(map[protobuf.Message]bool)
	var addFDsToSet func(fd getDependencyer)
	addFDsToSet = func(fd getDependencyer) {
		for _, depName := range fd.GetDependency() {
			depFD, ok := fdProtoStore.Load(depName)
			if ok {
				addFDsToSet(depFD.(getDependencyer))
			}
		}

		fdMsg := fd.(protobuf.Message)
		if _, ok := wrt.writtenFDs[fdMsg]; !ok {
			newFDs[fdMsg] = true
			wrt.writtenFDs[fdMsg] = true
		}
	}
	for _, typeName := range event.proto.Type {
		fdProto, ok := fdProtoForTypeStore.Load(typeName)
		if ok {
			addFDsToSet(fdProto.(getDependencyer))
		}
	}
	if len(newFDs) > 0 {
		wrt.Flush()
	}
	for fdProto := range newFDs {
		fdBytes, err := protobuf.Marshal(fdProto)
		if err != nil {
			return errors.New("Unable to marshal file descriptor proto")
		}
		wrt.bucketHeader.FileDescriptor = append(wrt.bucketHeader.FileDescriptor, fdBytes)
	}

	writeBytes(wrt.bucket, protoSizeBuf)
	writeBytes(wrt.bucket, protoBuf)

	wrt.bucketHeader.NEvents++

	if wrt.bucket.Len() > bucketDumpSize {
		if err := wrt.writeBucket(); err != nil {
			return err
		}
	}

	return nil
}

func (wrt *Writer) PushMetadata(name string, data []byte) error {
	if err := wrt.Flush(); err != nil {
		return err
	}
	if wrt.bucketHeader.Metadata == nil {
		wrt.bucketHeader.Metadata = make(map[string][]byte)
	}
	wrt.bucketHeader.Metadata[name] = data
	return nil
}

const bucketDumpSize = 0x1000000

var magicBytes = [...]byte{
	byte(0xe1),
	byte(0xc1),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
	byte(0x00),
}

func (wrt *Writer) writeBucket() error {
	bucketBytes := wrt.bucket.Bytes()
	switch wrt.bucketHeader.Compression {
	case proto.BucketHeader_GZIP:
		buffer := &bytes.Buffer{}
		gzipWriter := gzip.NewWriter(buffer)
		gzipWriter.Write(bucketBytes)
		gzipWriter.Close()
		bucketBytes = buffer.Bytes()
	case proto.BucketHeader_LZ4:
		buffer := &bytes.Buffer{}
		lz4Writer := lz4.NewWriter(buffer)
		lz4Writer.Write(bucketBytes)
		lz4Writer.Close()
		bucketBytes = buffer.Bytes()
	case proto.BucketHeader_LZMA:
		buffer := &bytes.Buffer{}
		lzmaWriter := lzma.NewWriter(buffer)
		lzmaWriter.Write(bucketBytes)
		lzmaWriter.Close()
		bucketBytes = buffer.Bytes()
	}
	header := wrt.bucketHeader
	header.BucketSize = uint64(len(bucketBytes))
	headerBuf, err := (&header).Marshal()
	if err != nil {
		return err
	}

	headerSizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(headerSizeBuf, uint32(len(headerBuf)))

	if err := writeBytes(wrt.streamWriter, magicBytes[:]); err != nil {
		return err
	}
	if err := writeBytes(wrt.streamWriter, headerSizeBuf); err != nil {
		return err
	}
	if err := writeBytes(wrt.streamWriter, headerBuf); err != nil {
		return err
	}
	if err := writeBytes(wrt.streamWriter, bucketBytes); err != nil {
		return err
	}

	wrt.bucketHeader.NEvents = 0
	wrt.bucketHeader.Metadata = make(map[string][]byte)
	wrt.bucketHeader.FileDescriptor = nil
	wrt.bucket.Reset()

	return nil
}

func writeBytes(wrt io.Writer, buf []byte) error {
	tot := 0
	for tot < len(buf) {
		n, err := wrt.Write(buf[tot:])
		tot += n
		if err != nil && tot != len(buf) {
			return err
		}
	}
	return nil
}

func (wrt *Writer) deferUntilClose(thisFunc func() error) {
	wrt.deferredUntilClose = append(wrt.deferredUntilClose, thisFunc)
}
