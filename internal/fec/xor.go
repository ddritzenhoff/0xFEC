package fec

import (
	"bytes"
	"fmt"
	"math"
	"sort"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/wire"
)

type Block struct {
	id                    protocol.BlockID
	params                blockParams
	biggestSourceLenSoFar int
	sidToSourceData       map[protocol.SID][]byte
	ridToRepairData       map[protocol.RID][]byte
}

func newBlock(id protocol.BlockID, params blockParams) *Block {
	return &Block{
		id:                    id,
		params:                params,
		sidToSourceData:       make(map[protocol.SID][]byte),
		biggestSourceLenSoFar: 0,
		ridToRepairData:       make(map[protocol.RID][]byte),
	}
}

func (b *Block) repairFrames() ([]*wire.RepairFrame, error) {
	if len(b.sidToSourceData) == b.params.k {
		return b.GetRepairSymbols()
	}
	return nil, nil
}

func (b *Block) data() ([]byte, error) {
	if len(b.sidToSourceData) == b.params.k {
		// we have all the needed source symbols, so we can just combine them.
		return b.combineSourceSymbols(), nil
	} else if len(b.sidToSourceData)+len(b.ridToRepairData) >= b.params.k {
		if err := b.RecoverSymbols(); err != nil {
			return nil, err
		}
		return b.combineSourceSymbols(), nil
	}
	return nil, fmt.Errorf("fec block is not full")
}

func (b *Block) addSourceSymbol(f *wire.SourceSymbolFrame) (isBlockFull bool) {
	if _, ok := b.sidToSourceData[f.SID]; !ok {
		b.sidToSourceData[f.SID] = f.Payload
		if b.biggestSourceLenSoFar < len(f.Payload) {
			b.biggestSourceLenSoFar = len(f.Payload)
		}
	}
	// The block is full when there are enough source symbols to satisfy k in the (n, k) block encoding scheme. Alternatively, the block is also full when the number of source symbols + the number of repair symbols are equal or greater than k.
	return len(b.sidToSourceData) == b.params.k || len(b.sidToSourceData)+len(b.ridToRepairData) >= b.params.k
}

func (b *Block) combineSourceSymbols() []byte {
	// Get and sort the keys.
	sids := make([]protocol.SID, 0, len(b.sidToSourceData))
	for sid := range b.sidToSourceData {
		sids = append(sids, sid)
	}
	sort.Slice(sids, func(i, j int) bool { return sids[i] < sids[j] })

	// Combine the source symbols in order.
	combined := make([]byte, 0, b.params.k*protocol.MaxPacketBufferSize)
	for _, sid := range sids {
		combined = append(combined, b.sidToSourceData[sid]...)
	}

	return combined
}

func (b *Block) RecoverSymbols() error {
	// if you manage to recover the symbols, the block will be complete. At that point, you would be able to hand over the data to receive API.
	if len(b.sidToSourceData)+len(b.ridToRepairData) < int(b.params.k) {
		return fmt.Errorf("not enough source and/or repair symbols to recover the remaining source symbols. %d source symbols and %d repair symbols in a (%d, %d) scheme", len(b.sidToSourceData), len(b.ridToRepairData), b.params.n, b.params.n)
	}

	if len(b.sidToSourceData) == int(b.params.k) {
		// the block already has all the source symbols, so there's nothing for us to do.
		return nil
	}

	recoveredSymbol := make([]byte, b.biggestSourceLenSoFar)
	for _, data := range b.ridToRepairData {
		b.xor(recoveredSymbol, data)
	}
	for _, data := range b.sidToSourceData {
		b.xor(recoveredSymbol, data)
	}

	// at this point, the symbol should be recovered. We just have to trim the extra zeros that may be hanging at the end.
	recoveredFrame, err := wire.ParseSourceSymbolFrame(bytes.NewReader(recoveredSymbol), protocol.VersionUnknown)
	if err != nil {
		return err
	}

	data, exists := b.sidToSourceData[recoveredFrame.SID]
	if exists {
		return fmt.Errorf("recovered symbol already exists within the block: sid: %d", recoveredFrame.SID)
	}
	b.sidToSourceData[recoveredFrame.SID] = data
	return nil
}

func (b *Block) GetRepairSymbols() ([]*wire.RepairFrame, error) {
	if b.params.n-b.params.k != 1 {
		return nil, fmt.Errorf("xor only supports a (k+1,k) scheme. provided (%d, %d)", b.params.n, b.params.k)
	}

	if len(b.sidToSourceData) != int(b.params.k) {
		return nil, fmt.Errorf("expecting %d source symbols. provided %d", b.params.k, len(b.sidToSourceData))
	}

	// Create a zero-slice the size of of the biggest SOURCE_SYMBOL frame, as that will determine the size of the REPAIR frame.
	xorSoFar := make([]byte, b.biggestSourceLenSoFar)
	smallestSID := protocol.SID(math.MaxUint64)
	largestSID := protocol.SID(0)
	for sid, data := range b.sidToSourceData {
		smallestSID = min(smallestSID, sid)
		largestSID = max(largestSID, sid)
		xorSoFar = b.xor(xorSoFar, data)
	}

	rid, err := protocol.NewRID(smallestSID, largestSID)
	if err != nil {
		return nil, err
	}

	return []*wire.RepairFrame{wire.NewRepairFrame(rid, b.id, xorSoFar)}, nil
}

// TODO (ddritzenhoff) this is the slow way of doing XOR, so you'll want to eventually add the faster version.
func (b *Block) xor(xorSoFar []byte, data []byte) []byte {
	for i := 0; i < len(data); i++ {
		xorSoFar[i] = xorSoFar[i] ^ data[i]
	}

	return xorSoFar
}
