package wire

import (
	"bytes"
	"io"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/quicvarint"
)

type RepairFrame struct {
	Metadata protocol.BlockMetadata
	Payload  []byte
}

func parseRepairFrame(r *bytes.Reader, _ protocol.Version) (*RepairFrame, error) {
	frame := &RepairFrame{}
	blockID, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	frame.Metadata.BlockID = protocol.BlockID(blockID)
	parityID, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	frame.Metadata.ParityID = protocol.ParityID(parityID)
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
			// this should never happen since we already checked the dataLen earlier.
			return nil, err
		}
	}
	return frame, nil
}

func (f *RepairFrame) Append(b []byte, _ protocol.Version) ([]byte, error) {
	b = quicvarint.Append(b, uint64(repairFrameType))
	b = quicvarint.Append(b, uint64(f.Metadata.BlockID))
	b = quicvarint.Append(b, uint64(f.Metadata.ParityID))
	b = quicvarint.Append(b, uint64(len(f.Payload)))
	b = append(b, f.Payload...)
	return b, nil
}

// Length
func (f *RepairFrame) Length(_ protocol.Version) protocol.ByteCount {
	return quicvarint.Len(uint64(repairFrameType)) + quicvarint.Len(uint64(f.Metadata.BlockID)) + quicvarint.Len(uint64(f.Metadata.ParityID)) + quicvarint.Len(uint64(len(f.Payload))) + protocol.ByteCount(len(f.Payload))
}
