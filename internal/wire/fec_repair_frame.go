package wire

import (
	"bytes"

	"github.com/quic-go/quic-go/internal/protocol"
)

type RepairFrame struct {
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
