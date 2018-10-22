package proio

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/pierrec/lz4"
	proto "github.com/proio-org/go-proio-pb"
	"github.com/smira/lzma"
)

// Reader serves to read Events from a stream in the proio format.  The Reader
// is not inherently thread safe, but it conveniently embeds sync.Mutex so that
// it can be locked and unlocked.
type Reader struct {
	BucketHeader *proto.BucketHeader
	Metadata     map[string][]byte
	Err          error

	streamReader          io.Reader
	bucket                *bytes.Reader
	bucketReader          io.Reader
	bucketEventsRead      uint64
	bucketIndex           uint64
	deferredUntilStopScan []func()
	deferredUntilClose    []func()

	sync.Mutex
}

// Open opens the given existing file (in read-only mode), returning an error
// where appropriate.  Upon success, a new Reader is created to wrap the file,
// and returned.  Either Open or NewReader should be called to construct a new
// Reader.
func Open(filename string) (*Reader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	reader := NewReader(file)
	reader.deferUntilClose(func() { file.Close() })
	return reader, nil
}

// NewReader wraps an existing io.Reader for reading proio Events.  Either Open
// or NewReader should be called to construct a new Reader.
func NewReader(streamReader io.Reader) *Reader {
	rdr := &Reader{
		Metadata:     make(map[string][]byte),
		streamReader: streamReader,
		bucket:       &bytes.Reader{},
		bucketReader: &bytes.Buffer{},
	}

	return rdr
}

// Close closes any file that was opened by the library, and stops any
// unfinished scans.  Close does not close io.Readers passed directly to
// NewReader.
func (rdr *Reader) Close() {
	rdr.StopScan()
	for _, thisFunc := range rdr.deferredUntilClose {
		thisFunc()
	}
	rdr.deferredUntilClose = nil
}

// Next retrieves the next event from the stream.  The Reader's Err member is
// assigned the error status of this call.
func (rdr *Reader) Next() *Event {
	var event *Event

	// use Skip() to ensure that we land on a non-empty bucket
	if _, rdr.Err = rdr.Skip(0); rdr.Err == nil {
		if rdr.bucket.Size() == 0 {
			rdr.readBucket()
		}

		event, rdr.Err = rdr.readFromBucket()
	}

	return event
}

// Skip skips nEvents events.  If the return error is nil, nEvents have been
// skipped.
func (rdr *Reader) Skip(nEvents uint64) (nSkipped uint64, err error) {
	startIndex := rdr.bucketIndex
	rdr.bucketIndex += nEvents

	// loop while until reaching a bucket header that describes a bucket
	// containg the event being skipped to
	for rdr.BucketHeader == nil || rdr.bucketIndex >= rdr.BucketHeader.NEvents {
		if rdr.BucketHeader != nil {
			nBucketEvents := rdr.BucketHeader.NEvents
			rdr.bucketIndex -= nBucketEvents
			nSkipped += nBucketEvents - startIndex

			// skip the bucket bytes on the stream if they haven't been read
			// into memory already
			if nBucketEvents > 0 && rdr.bucket.Size() == 0 {
				seeker, ok := rdr.streamReader.(io.Seeker)
				if ok {
					seekBytes(seeker, int64(rdr.BucketHeader.BucketSize))
				} else {
					bucketBytes := make([]byte, rdr.BucketHeader.BucketSize)
					if err = readBytes(rdr.streamReader, bucketBytes); err != nil {
						return
					}
				}
			}
		}

		err = rdr.readHeader()
		if rdr.BucketHeader == nil {
			return
		}
		startIndex = 0
	}
	nSkipped += rdr.bucketIndex - startIndex

	return
}

// SeekToStart seeks seekable streams to the beginning, and prepares the stream
// to read from there.
func (rdr *Reader) SeekToStart() error {
	seeker, ok := rdr.streamReader.(io.Seeker)
	if !ok {
		return errors.New("stream not seekable")
	}

	for {
		n, err := seeker.Seek(0, 0 /*io.SeekStart*/)
		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
	}

	rdr.Metadata = make(map[string][]byte)
	rdr.bucketIndex = 0
	if err := rdr.readHeader(); err != nil {
		return err
	}

	return nil
}

// ScanEvents returns a buffered channel of type Event where all of the events
// in the stream will be pushed.  The channel buffer size is defined by the
// argument.  The goroutine responsible for fetching events will not break
// until there are no more events, Reader.StopScan() is called, or
// Reader.Close() is called.
func (rdr *Reader) ScanEvents(bufSize int) <-chan *Event {
	events := make(chan *Event, bufSize)
	quit := make(chan int)

	rdr.deferUntilStopScan(
		func() {
			close(quit)
		},
	)

	go func() {
		defer close(events)

		for {
			rdr.Lock()
			event := rdr.Next()
			rdr.Unlock()
			if event == nil {
				return
			}

			select {
			case events <- event:
			case <-quit:
				return
			}
		}
	}()

	return events
}

// StopScan stops all scans initiated by Reader.ScanEvents().
func (rdr *Reader) StopScan() {
	for _, thisFunc := range rdr.deferredUntilStopScan {
		thisFunc()
	}
	rdr.deferredUntilStopScan = nil
}

func (rdr *Reader) deferUntilClose(thisFunc func()) {
	rdr.deferredUntilClose = append(rdr.deferredUntilClose, thisFunc)
}

func (rdr *Reader) deferUntilStopScan(thisFunc func()) {
	rdr.deferredUntilStopScan = append(rdr.deferredUntilStopScan, thisFunc)
}

func (rdr *Reader) readFromBucket() (*Event, error) {
	var event *Event

	for rdr.bucketEventsRead <= rdr.bucketIndex {
		protoSizeBuf := make([]byte, 4)
		if err := readBytes(rdr.bucketReader, protoSizeBuf); err != nil {
			return nil, err
		}

		protoSize := binary.LittleEndian.Uint32(protoSizeBuf)

		protoBuf := make([]byte, protoSize)
		if err := readBytes(rdr.bucketReader, protoBuf); err != nil {
			return nil, err
		}

		if rdr.bucketEventsRead == rdr.bucketIndex {
			eventProto := &proto.Event{}
			if err := eventProto.Unmarshal(protoBuf); err != nil {
				return nil, err
			}

			event = newEventFromProto(eventProto)
			for key, bytes := range rdr.Metadata {
				event.Metadata[key] = bytes
			}
		}

		rdr.bucketEventsRead++
	}
	rdr.bucketIndex++

	return event, nil
}

func (rdr *Reader) readHeader() (err error) {
	rdr.bucketEventsRead = 0
	rdr.BucketHeader = nil
	rdr.bucket = &bytes.Reader{}

	// Find and read magic bytes for synchronization
	var n int
	n, err = rdr.syncToMagic()
	if err != nil {
		return
	}

	// Read header size and then header
	headerSizeBuf := make([]byte, 4)
	if err = readBytes(rdr.streamReader, headerSizeBuf); err != nil {
		return
	}
	headerSize := binary.LittleEndian.Uint32(headerSizeBuf)

	headerBuf := make([]byte, headerSize)
	if err = readBytes(rdr.streamReader, headerBuf); err != nil {
		return
	}
	bucketHeader := &proto.BucketHeader{}
	if err = bucketHeader.Unmarshal(headerBuf); err != nil {
		return
	}
	rdr.BucketHeader = bucketHeader

	// Set metadata for future events
	for key, bytes := range rdr.BucketHeader.Metadata {
		rdr.Metadata[key] = bytes
	}

	// Add descriptors to pool
	for _, fdBytes := range rdr.BucketHeader.FileDescriptor {
		if err = addFDFromBytes(fdBytes); err != nil {
			return
		}
	}

	if n != len(magicBytes) {
		return errors.New("stream resynchronized")
	}
	return
}

func (rdr *Reader) readBucket() (err error) {
	// read bucket bytes
	bucketBytes := make([]byte, rdr.BucketHeader.BucketSize)
	if err = readBytes(rdr.streamReader, bucketBytes); err != nil {
		return
	}
	rdr.bucket.Reset(bucketBytes)

	// Set up decompression for bucket
	switch rdr.BucketHeader.Compression {
	case proto.BucketHeader_GZIP:
		gzipRdr, ok := rdr.bucketReader.(*gzip.Reader)
		if ok {
			gzipRdr.Reset(rdr.bucket)
		} else {
			gzipRdr, err = gzip.NewReader(rdr.bucket)
			if err != nil {
				return
			}
		}
		rdr.bucketReader = gzipRdr
	case proto.BucketHeader_LZ4:
		lz4Rdr, ok := rdr.bucketReader.(*lz4.Reader)
		if ok {
			lz4Rdr.Reset(rdr.bucket)
		} else {
			lz4Rdr = lz4.NewReader(rdr.bucket)
		}
		rdr.bucketReader = lz4Rdr
	case proto.BucketHeader_LZMA:
		lzmaRdr := lzma.NewReader(rdr.bucket)
		rdr.bucketReader = lzmaRdr
	case proto.BucketHeader_NONE:
		rdr.bucketReader = rdr.bucket
	default:
		err = errors.New("unknown bucket compression type")
	}

	return
}

func (rdr *Reader) syncToMagic() (int, error) {
	magicByteBuf := make([]byte, 1)
	nRead := 0
	for {
		err := readBytes(rdr.streamReader, magicByteBuf)
		if err != nil {
			return nRead, err
		}
		nRead++

		if magicByteBuf[0] == magicBytes[0] {
			var goodSeq = true
			for i := 1; i < len(magicBytes); i++ {
				err := readBytes(rdr.streamReader, magicByteBuf)
				if err != nil {
					return nRead, err
				}
				nRead++

				if magicByteBuf[0] != magicBytes[i] {
					goodSeq = false
					break
				}
			}

			if goodSeq {
				break
			}
		}
	}

	return nRead, nil
}

func readBytes(rdr io.Reader, buf []byte) error {
	tot := 0
	for tot < len(buf) {
		n, err := rdr.Read(buf[tot:])
		tot += n
		if err != nil && tot != len(buf) {
			return err
		}
	}
	return nil
}

func seekBytes(seeker io.Seeker, nBytes int64) error {
	start, err := seeker.Seek(0, 1 /*io.SeekCurrent*/)
	if err != nil {
		return err
	}

	tot := int64(0)
	for tot < nBytes {
		n, err := seeker.Seek(int64(nBytes-tot), 1 /*io.SeekCurrent*/)
		tot += n - start
		if err != nil && tot != nBytes {
			return err
		}
	}
	return nil
}
