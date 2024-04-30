package wire

import (
	"bytes"

	"github.com/quic-go/quic-go/internal/protocol"
)

/*
You basically have to extract the frames and then send them back to frame_parser. You also have to think of edge cases like a source symbol being within another source symbol.
*/

type SourceSymbolFrame struct {
}

// TODO (ddritzenhoff) finish implementing.
func parseSourceSymbolFrame(_ *bytes.Reader, _ protocol.Version) (*SourceSymbolFrame, error) {
	return nil, nil
}

func (f *SourceSymbolFrame) Append(b []byte, _ protocol.Version) ([]byte, error) {
	return nil, nil
}

// Length of a written frame
func (f *SourceSymbolFrame) Length(_ protocol.Version) protocol.ByteCount {
	return 0
}
