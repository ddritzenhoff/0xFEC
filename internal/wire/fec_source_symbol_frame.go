package wire

import (
	"bytes"
	"io"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/quicvarint"
)

type SourceSymbolFrame struct {
	SID        protocol.SID
	PayloadLen protocol.ByteCount
	Payload    []byte
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

func (f *SourceSymbolFrame) Append(b []byte, _ protocol.Version) ([]byte, error) {
	b = quicvarint.Append(b, uint64(sourceSymbolFrameType))
	b = quicvarint.Append(b, uint64(f.SID))
	b = quicvarint.Append(b, uint64(f.PayloadLen))
	b = append(b, f.Payload...)
	return b, nil
}

// Length of a written frame
func (f *SourceSymbolFrame) Length(_ protocol.Version) protocol.ByteCount {
	return quicvarint.Len(uint64(sourceSymbolFrameType)) + quicvarint.Len(uint64(f.SID)) + quicvarint.Len(uint64(f.PayloadLen)) + protocol.ByteCount(len(f.Payload))
}
