package wire

import (
	"bytes"
	"io"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/quicvarint"
)

type RepairFrame struct {
	// Metadata should identify the source symbols protected by the repair symbol carried by the frame
	/*
		- Another thing to think about is whether the repair frame contains just one repair symbol or multiple. Currently, I'm not sure.
		- "Depending on the FEC scheme, the REPAIR frame may contain only a part of a repair symbol."
		- Dive deeper into interacting with congestion control algorithms. Within Francois' QUIC-FEC section, he mentioned that a lost-and-recovered packet should be considered as a lost packet. My question then is whether that'll lead to a waste in bandwidth as the sender will try to re-send the packet? I suppose that's what the symbol-ack-frame is for. The loss will be registered by the congestion control algorithm, but no retransmissions will be initiated.
	*/
	RID  protocol.RID
	Data []byte
}

func NewRepairFrame(RID protocol.RID, Data []byte) *RepairFrame {
	return &RepairFrame{
		RID,
		Data,
	}
}

func parseRepairFrame(r *bytes.Reader, _ protocol.Version) (*RepairFrame, error) {
	frame := &RepairFrame{}
	smallestSID, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	frame.RID.SmallestSID = protocol.SID(smallestSID)
	highestSID, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	frame.RID.LargestSID = protocol.SID(highestSID)
	dataLen, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	if dataLen > uint64(r.Len()) {
		return nil, io.EOF
	}
	if dataLen != 0 {
		frame.Data = make([]byte, dataLen)
		if _, err := io.ReadFull(r, frame.Data); err != nil {
			// this should never happen, since we already checked the dataLen earlier.
			return nil, err
		}
	}
	return frame, nil
}

func (f *RepairFrame) Append(b []byte, _ protocol.Version) ([]byte, error) {
	b = quicvarint.Append(b, uint64(repairFrameType))
	b = quicvarint.Append(b, uint64(f.RID.SmallestSID))
	b = quicvarint.Append(b, uint64(f.RID.LargestSID))
	b = quicvarint.Append(b, uint64(len(f.Data)))
	b = append(b, f.Data...)
	return b, nil
}

func (f *RepairFrame) MaxHeaderLen() protocol.ByteCount {
	return quicvarint.Len(uint64(repairFrameType)) + 2 + 2 + 2
}

// Length of a written frame
func (f *RepairFrame) Length(_ protocol.Version) protocol.ByteCount {
	return quicvarint.Len(uint64(repairFrameType)) + quicvarint.Len(uint64(f.RID.SmallestSID)) + quicvarint.Len(uint64(f.RID.LargestSID)) + quicvarint.Len(uint64(len(f.Data))) + protocol.ByteCount(len(f.Data))
}
