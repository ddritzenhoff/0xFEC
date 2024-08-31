package wire

import (
	"bytes"
	"io"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/quicvarint"
)

type SourceSymbolFrame struct {
	SID protocol.SID
	// TODO (ddritzenhoff) I wonder if I can make this more efficient by pre-allocating a size when I create the frame for the first time.
	Payload []byte
}

func NewSourceSymbolFrame(SID protocol.SID, Payload []byte) *SourceSymbolFrame {
	return &SourceSymbolFrame{
		SID:     SID,
		Payload: Payload,
	}
}

func ParseSourceSymbolFrame(r *bytes.Reader, _ protocol.Version) (*SourceSymbolFrame, error) {
	frame := &SourceSymbolFrame{}
	sid, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	frame.SID = protocol.SID(sid)
	payloadLen, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	if payloadLen > uint64(r.Len()) {
		return nil, io.EOF
	}
	if payloadLen != 0 {
		frame.Payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(r, frame.Payload); err != nil {
			// this should never happen since we already checked the payloadLen earlier.
			return nil, err
		}
	}
	return frame, nil
}

func (f *SourceSymbolFrame) HeadLen() protocol.ByteCount {
	return quicvarint.Len(uint64(sourceSymbolFrameType)) + quicvarint.Len(uint64(f.SID)) + quicvarint.Len(uint64(len(f.Payload)))
}

func (f *SourceSymbolFrame) MaxHeaderLen() protocol.ByteCount {
	// Realistically, SID will never be anything more than 2 bytes, and the payload Length will never be anything more 4 bytes.
	return quicvarint.Len(uint64(sourceSymbolFrameType)) + 2 + 4
}

func (f *SourceSymbolFrame) Append(b []byte, v protocol.Version) ([]byte, error) {
	b = quicvarint.Append(b, uint64(sourceSymbolFrameType))
	b = quicvarint.Append(b, uint64(f.SID))
	b = quicvarint.Append(b, uint64(len(f.Payload)))
	b = append(b, f.Payload...)
	return b, nil
}

// Length of a written frame
func (f *SourceSymbolFrame) Length(_ protocol.Version) protocol.ByteCount {
	return quicvarint.Len(uint64(sourceSymbolFrameType)) + quicvarint.Len(uint64(f.SID)) + quicvarint.Len(uint64(len(f.Payload))) + protocol.ByteCount(len(f.Payload))
}
