package fec

import (
	"bytes"
	"fmt"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/wire"
)

type XORScheme struct {
}

func (s *XORScheme) RecoverSymbols(block *Block) error {
	// if you manage to recover the symbols, the block will be complete. At that point, you would be able to hand over the data to receive API.
	if len(block.sidToSourceData)+len(block.ridToRepairData) < int(block.scheme.k) {
		return fmt.Errorf("not enough source and/or repair symbols to recover the remaining source symbols. %d source symbols and %d repair symbols in a (%d, %d) scheme", len(block.sidToSourceData), len(block.ridToRepairData), block.scheme.n, block.scheme.n)
	}

	if len(block.sidToSourceData) == int(block.scheme.k) {
		// the block already has all the source symbols, so there's nothing for us to do.
		return nil
	}

	recoveredSymbol := make([]byte, block.biggestSourceLenSoFar)
	for _, data := range block.ridToRepairData {
		s.xor(recoveredSymbol, data)
	}
	for _, data := range block.sidToSourceData {
		s.xor(recoveredSymbol, data)
	}

	// at this point, the symbol should be recovered. We just have to trim the extra zeros that may be hanging at the end.
	recoveredFrame, err := wire.ParseSourceSymbolFrame(bytes.NewReader(recoveredSymbol), protocol.VersionUnknown)
	if err != nil {
		return err
	}

	data, exists := block.sidToSourceData[recoveredFrame.SID]
	if exists {
		return fmt.Errorf("recovered symbol already exists within the block: sid: %d", recoveredFrame.SID)
	}
	block.sidToSourceData[recoveredFrame.SID] = data
	return nil
}

func (s *XORScheme) GetRepairSymbols(block *Block, numberOfSymbols uint) ([]*wire.RepairFrame, error) {
	if block.scheme.n-block.scheme.k != 1 {
		return nil, fmt.Errorf("xor only supports a (k+1,k) scheme. provided (%d, %d)", block.scheme.n, block.scheme.k)
	}

	if len(block.sidToSourceData) != int(block.scheme.k) {
		return nil, fmt.Errorf("expecting %d source symbols. provided %d", block.scheme.k, len(block.sidToSourceData))
	}

	// TODO (ddritzenhoff) assuming block.biggestSoFar can't exceed the maximum frame size. protocol.MaxPacketBufferSize
	xorSoFar := make([]byte, block.biggestSourceLenSoFar)
	for i := range xorSoFar {
		xorSoFar[i] = 0
	}

	for _, data := range block.sidToSourceData {
		xorSoFar = s.xor(xorSoFar, data)
	}

	return []*wire.RepairFrame{wire.NewRepairFrame(block.generateRID(), block.id, xorSoFar)}, nil
}

// TODO (ddritzenhoff) this is the slow way of doing XOR, so you'll want to eventually add the faster version.
func (s *XORScheme) xor(xorSoFar []byte, data []byte) []byte {
	for i := 0; i < len(data); i++ {
		xorSoFar[i] = xorSoFar[i] ^ data[i]
	}

	return xorSoFar
}
