package wire

import (
	"bytes"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/quicvarint"
)

type FECWindowFrame struct {
	// TODO (dditzenhoff) Do these type sizes make sense? I'm pretty sure that we'll never get to an epoch size above 2^16= ~65,000 and a window size above 4GB. TCP Window scaling only goes up to 1 GiB, for example.
	Epoch protocol.FECWindowEpoch
	Size  protocol.FECWindowSize
}

func parseFECWindowFrame(r *bytes.Reader, _ protocol.Version) (*FECWindowFrame, error) {
	f := &FECWindowFrame{}
	epoch, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	f.Epoch = protocol.FECWindowEpoch(epoch)
	size, err := quicvarint.Read(r)
	if err != nil {
		return nil, err
	}
	f.Size = protocol.FECWindowSize(size)
	return f, nil
}

func (f *FECWindowFrame) Append(b []byte, _ protocol.Version) ([]byte, error) {
	b = quicvarint.Append(b, uint64(FECWindowFrameType))
	b = quicvarint.Append(b, uint64(f.Epoch))
	b = quicvarint.Append(b, uint64(f.Size))

	return b, nil
}

func (f *FECWindowFrame) Length(protocol.Version) protocol.ByteCount {
	return quicvarint.Len(FECWindowFrameType) + quicvarint.Len(uint64(f.Epoch)) + quicvarint.Len(uint64(f.Size))
}
