package wire

import (
	"bytes"

	"github.com/quic-go/quic-go/internal/protocol"
)

type RepairFrame struct {
	// Metadata should identify the source symbols protected by the repair symbol carried by the frame
	/*
		- Another thing to think about is whether the repair frame contains just one repair symbol or multiple. Currently, I'm not sure.
		- "Depending on the FEC scheme, the REPAIR frame may contain only a part of a repair symbol."
		- Dive deeper into interacting with congestion control algorithms. Within Francois' QUIC-FEC section, he mentioned that a lost-and-recovered packet should be considered as a lost packet. My question then is whether that'll lead to a waste in bandwidth as the sender will try to re-send the packet? I suppose that's what the symbol-ack-frame is for. The loss will be registered by the congestion control algorithm, but no retransmissions will be initiated.
	*/
	RID     protocol.RID
	BlockID protocol.BlockID
	Data    []byte
}

func NewRepairFrame(RID protocol.RID, BlockID protocol.BlockID, Data []byte) *RepairFrame {
	return &RepairFrame{
		RID,
		BlockID,
		Data,
	}
}

// TODO (ddritzenhoff) finish implementing.
func parseRepairFrame(_ *bytes.Reader, _ protocol.Version) (*RepairFrame, error) {
	return nil, nil
}

func (f *RepairFrame) Append(b []byte, _ protocol.Version) ([]byte, error) {
	return nil, nil
}

// Length of a written frame
func (f *RepairFrame) Length(_ protocol.Version) protocol.ByteCount {
	return 0
}
