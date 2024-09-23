package fec

import (
	"fmt"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/wire"
)

type BlockScheme interface {
	// addSourceSymbol adds a source symbol to the block. An error is thrown if it gets a source symbol with a SID outside of what the block is meant to protect.
	addSourceSymbol(f *wire.SourceSymbolFrame) error
	// processSourceSymbol marks this particular source symbol as already having been passed up to the application.
	processSourceSymbol(f *wire.SourceSymbolFrame)
	// addRepairSymbol adds a repair symbol to the block. An error is thrown if it gets a repair symbol with SIDs outside of what the block is meant to protect.
	addRepairSymbol(f *wire.RepairFrame) error
	// isRecoverable indicates whether a block contains enough repair and source symbols to repair missing source symbols.
	isRecoverable() bool
	// isComplete indicates whether a block contains all of its source symbols.
	isComplete() bool
}

type block struct {
	id                  protocol.BlockID
	ssidToSourcePayload map[protocol.SourceSymbolID][]byte
	pidToRepairPayload  map[protocol.ParityID][]byte
	smallestSSID        protocol.SourceSymbolID
	largestSSID         protocol.SourceSymbolID
	// totNumSourceSymbols represents the total number of source symbols in this block.
	totNumSourceSymbols int
	// totNumRepairSymbols represents the total number of repair symbols in this block.
	totNumRepairSymbols         int
	biggestSourceSymbolLenSoFar int
}

func newBlock(id protocol.BlockID, totNumSourceSymbols int, totNumRepairSymbols int) *block {
	smallestSSID := protocol.SourceSymbolID(id) * protocol.SourceSymbolID(totNumSourceSymbols)
	largestSSID := smallestSSID + protocol.SourceSymbolID(totNumSourceSymbols) - 1
	return &block{
		id:                          id,
		ssidToSourcePayload:         make(map[protocol.SourceSymbolID][]byte),
		pidToRepairPayload:          make(map[protocol.ParityID][]byte),
		smallestSSID:                smallestSSID,
		largestSSID:                 largestSSID,
		totNumSourceSymbols:         int(largestSSID) - int(smallestSSID) + 1,
		totNumRepairSymbols:         totNumRepairSymbols,
		biggestSourceSymbolLenSoFar: 0,
	}
}

/*
TODO: Look the lengths of created source symbols. They should ideally all have a capacity of protocol.MaxBufferLength (1452). I assume what'll mostly happen is that a gross majority of the packets will contain a single stream frame that reaches its max possible size. The very last stream frame will then be small due to the remaining data.
*/

// addSourceSymbol adds a source symbol to the block. An error is thrown if it gets a source symbol with a SID outside of what the block is meant to protect.
func (b *block) addSourceSymbol(f *wire.SourceSymbolFrame) error {
	if f.SSID < b.smallestSSID || f.SSID > b.largestSSID {
		return fmt.Errorf("source symbol was provided to the wrong block. Expecting SID within the range [%d, %d] and got %d", b.smallestSSID, b.largestSSID, f.SSID)
	}

	// at this point, we know the source symbol belongs to the block.

	if _, exists := b.ssidToSourcePayload[f.SSID]; !exists {
		b.ssidToSourcePayload[f.SSID] = f.Payload
		if b.biggestSourceSymbolLenSoFar < len(f.Payload) {
			b.biggestSourceSymbolLenSoFar = len(f.Payload)
		}
	}
	return nil
}

// addRepairSymbol adds a repair symbol to the block. An error is thrown if it gets a repair symbol with SIDs outside of what the block is meant to protect.
func (b *block) addRepairSymbol(f *wire.RepairFrame) error {
	if b.id != f.Metadata.BlockID {
		return fmt.Errorf("the repair symbol was provided to the wrong block. Expecting %d and got %d", b.id, f.Metadata.BlockID)
	}

	// at this point, we know the repair symbol belongs to the block

	if _, exists := b.pidToRepairPayload[f.Metadata.ParityID]; !exists {
		b.pidToRepairPayload[f.Metadata.ParityID] = f.Payload
		b.biggestSourceSymbolLenSoFar = len(f.Payload) - protocol.RepairPayloadMetadataLen
	}
	return nil
}

// isRecoverable indicates whether a block is 'full' in that it contains all its source symbols or it containts enough repair symbols and source symbols to repair missing source symbols.
func (b *block) isRecoverable() bool {
	return len(b.ssidToSourcePayload)+len(b.pidToRepairPayload) >= b.totNumSourceSymbols
}

// isComplete indicates whether a block contains all of its source symbols.
func (b *block) isComplete() bool {
	return len(b.ssidToSourcePayload) == b.totNumSourceSymbols
}
