package wire

import (
	"sync"

	"github.com/quic-go/quic-go/internal/protocol"
)

var pool sync.Pool

func init() {
	pool.New = func() interface{} {
		return &StreamFrame{
			Data:     make([]byte, 0, protocol.MaxPacketBufferSize),
			fromPool: true,
		}
	}
}

func GetStreamFrame() *StreamFrame {
	f := pool.Get().(*StreamFrame)
	// This ensures a stream frame will never be retrieved in which FECProtected is set to true.
	f.FECProtected = false
	return f
}

func putStreamFrame(f *StreamFrame) {
	if !f.fromPool {
		return
	}
	if protocol.ByteCount(cap(f.Data)) != protocol.MaxPacketBufferSize {
		panic("wire.PutStreamFrame called with packet of wrong size!")
	}
	pool.Put(f)
}
