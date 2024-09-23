package wire

import (
	"bytes"
	"io"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/quicvarint"
)

type SourceSymbolFrame struct {
	// SSID represents the source symbol ID. Each source symbol ID is unique.
	SSID protocol.SourceSymbolID
	// Payload represents a collection of STREAM and DATAGRAM frames.
	// TODO (ddritzenhoff) I wonder if I can make this more efficient by pre-allocating a size when I create the frame for the first time.
	Payload []byte
}

func ParseSourceSymbolFrame(r *bytes.Reader, _ protocol.Version) (*SourceSymbolFrame, error) {
	frame := &SourceSymbolFrame{}
	sid, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	frame.SSID = protocol.SourceSymbolID(sid)
	payloadLen, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	if payloadLen > uint64(r.Len()) {
		return nil, io.EOF
	}
	if payloadLen != 0 {
		frame.Payload = make([]byte, payloadLen, protocol.MaxPacketBufferSize)
		if _, err := io.ReadFull(r, frame.Payload); err != nil {
			// this should never happen since we already checked the payloadLen earlier.
			return nil, err
		}
	}
	return frame, nil
}

func (f *SourceSymbolFrame) HeaderLen() protocol.ByteCount {
	return quicvarint.Len(uint64(sourceSymbolFrameType)) + quicvarint.Len(uint64(f.SSID)) + quicvarint.Len(uint64(len(f.Payload)))
}

func (f *SourceSymbolFrame) Append(b []byte, v protocol.Version) ([]byte, error) {
	b = quicvarint.Append(b, uint64(sourceSymbolFrameType))
	b = quicvarint.Append(b, uint64(f.SSID))
	b = quicvarint.Append(b, uint64(len(f.Payload)))
	b = append(b, f.Payload...)
	return b, nil
}

// Length of a written frame
func (f *SourceSymbolFrame) Length(_ protocol.Version) protocol.ByteCount {
	return quicvarint.Len(uint64(sourceSymbolFrameType)) + quicvarint.Len(uint64(f.SSID)) + quicvarint.Len(uint64(len(f.Payload))) + protocol.ByteCount(len(f.Payload))
}

/*
TODO (ddritzenhoff) consider adding a pool for source symbols. On the sender side, you're creating source symbols and storing them in the block without doing anything else. I assume you'll converge on a certain pool eventually.

var pool sync.Pool

func init() {
	pool.New = func() interface{} {
		return &SourceSymbol{
			Payload:     make([]byte, 0, protocol.MaxPacketBufferSize),
			fromPool: true,
		}
	}
}

func GetSourceSymbolFrame() *SourceSymbol {
	f := pool.Get().(*SourceSymbol)
	return f
}

func putSourceSymbolFrame(f *StreamFrame) {
	if !f.fromPool {
		return
	}
	if protocol.ByteCount(cap(f.Data)) != protocol.MaxPacketBufferSize {
		panic("wire.PutSourceSymbolFrame called with packet of wrong size!")
	}
	pool.Put(f)
}
*/
