package fec

import (
	"fmt"
	"sync"

	"github.com/klauspost/reedsolomon"
	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/wire"
)

// Sender represents sender-side functions.
type Sender interface {
	AddSourceSymbolFrame(f *wire.SourceSymbolFrame) ([]*wire.RepairFrame, error)
	NextSSID() protocol.SourceSymbolID
}

// Receiver represents receiver-side functions.
type Receiver interface {
	HandleRepairFrame(f *wire.RepairFrame) ([]byte, error)
	HandleSourceSymbolFrame(f *wire.SourceSymbolFrame) ([]byte, error)
}

type Manager interface {
	Sender
	Receiver

	// TODO (ddritzenhoff) implement at a later time.
	// HandleSymbolAckFrame()
	// HandleFECWindowFrame()
	// UpdateWindowSize(newSize protocol.FECWindowSize, epoch protocol.FECWindowEpoch) error
	// SetInitialCodingWindow(ws protocol.FECWindowSize)
}

type blockStatus struct {
	block *block
	// isProcessed represents whether all the source symbols within the block have been passed up to the application.
	isProcessed bool
}

type manager struct {
	scheme              BlockFECScheme
	nextSIDMutex        sync.Mutex
	nextSID             protocol.SourceSymbolID
	numTotSourceSymbols int
	numTotRepairSymbols int
	blockStatuses       map[protocol.BlockID]blockStatus
}

func NewSender(id protocol.DecoderFECScheme) (Sender, error) {
	switch id {
	case protocol.FECDisabled:
		return nil, nil
	case protocol.XORFECScheme:
		xorScheme := xorScheme{}
		return NewManager(&xorScheme, 2, 1)
	case protocol.ReedSolomonFECScheme:
		numTotSourceSymbols := 20
		numTotRepairSymbols := 10
		reedSolomonEncoder, err := reedsolomon.New(numTotSourceSymbols, numTotRepairSymbols)
		if err != nil {
			return nil, err
		}
		reedSolomonScheme := reedSolomonScheme{
			enc: reedSolomonEncoder,
		}
		return NewManager(&reedSolomonScheme, numTotSourceSymbols, numTotRepairSymbols)
	default:
		return nil, fmt.Errorf("unknown FEC scheme: %d", id)
	}
}

func NewReceiver(id protocol.DecoderFECScheme) (Receiver, error) {
	switch id {
	case protocol.FECDisabled:
		return nil, nil
	case protocol.XORFECScheme:
		xorScheme := xorScheme{}
		return NewManager(&xorScheme, 2, 1)
	case protocol.ReedSolomonFECScheme:
		numTotSourceSymbols := 20
		numTotRepairSymbols := 10
		reedSolomonEncoder, err := reedsolomon.New(numTotSourceSymbols, numTotRepairSymbols)
		if err != nil {
			return nil, err
		}
		reedSolomonScheme := reedSolomonScheme{
			enc: reedSolomonEncoder,
		}
		return NewManager(&reedSolomonScheme, numTotSourceSymbols, numTotRepairSymbols)
	default:
		return nil, fmt.Errorf("unknown FEC scheme: %d", id)
	}
}

func NewManager(scheme BlockFECScheme, numTotSourceSymbols int, numTotRepairSymbols int) (*manager, error) {
	if numTotSourceSymbols < 0 || numTotRepairSymbols < 0 {
		return nil, fmt.Errorf("numTotSourceSymbols (%d) and numTotRepairSymbols (%d) may not be negative", numTotSourceSymbols, numTotRepairSymbols)
	}

	return &manager{
		nextSID:             0,
		numTotSourceSymbols: numTotSourceSymbols,
		numTotRepairSymbols: numTotRepairSymbols,
		scheme:              scheme,

		blockStatuses: make(map[protocol.BlockID]blockStatus),
	}, nil
}

func (m *manager) NextSSID() protocol.SourceSymbolID {
	m.nextSIDMutex.Lock()
	ret := m.nextSID
	m.nextSID++
	m.nextSIDMutex.Unlock()
	return ret
}

func (m *manager) sidToBlockID(sid protocol.SourceSymbolID) protocol.BlockID {
	return protocol.BlockID(uint64(sid) / uint64(m.numTotSourceSymbols))
}

func (m *manager) AddSourceSymbolFrame(f *wire.SourceSymbolFrame) ([]*wire.RepairFrame, error) {
	blockID := m.sidToBlockID(f.SSID)
	if _, exists := m.blockStatuses[blockID]; !exists {
		m.blockStatuses[blockID] = blockStatus{
			block:       newBlock(blockID, m.numTotSourceSymbols, m.numTotRepairSymbols),
			isProcessed: false}
	}

	bS := m.blockStatuses[blockID]
	if bS.isProcessed {
		// we've already processed the block, so we can ignore this source symbol
		return nil, nil

	}

	err := bS.block.addSourceSymbol(f)
	if err != nil {
		return nil, err
	}

	// check if the block is complete, so we can generate repair frames.
	if bS.block.isComplete() {
		repairSymbols, err := m.scheme.repairSymbols(bS.block)
		if err != nil {
			return nil, err
		}

		// drop the block as you don't need it anymore
		bS.block = nil
		bS.isProcessed = true
		m.blockStatuses[blockID] = bS
		return repairSymbols, nil
	}
	m.blockStatuses[blockID] = bS
	return nil, nil
}

func (m *manager) HandleRepairFrame(f *wire.RepairFrame) ([]byte, error) {

	// It's possible a repair frame arrives before any of its associated source symbol frames in the case they were dropped.
	if _, exists := m.blockStatuses[f.Metadata.BlockID]; !exists {
		m.blockStatuses[f.Metadata.BlockID] = blockStatus{
			block:       newBlock(f.Metadata.BlockID, m.numTotSourceSymbols, m.numTotRepairSymbols),
			isProcessed: false,
		}
	}

	bS := m.blockStatuses[f.Metadata.BlockID]
	if bS.isProcessed {
		// we've already processed the block, so we can ignore this repair symbol
		return nil, nil
	}

	err := bS.block.addRepairSymbol(f)
	if err != nil {
		return nil, err
	}

	if bS.block.isRecoverable() {
		sourceSymbolBytes, err := m.scheme.recoverSymbolPayloads(bS.block)
		if err != nil {
			return nil, err
		}

		// at this point, we've recovered all of the missing source symbols, which makes the block complete (i.e. processed)

		bS.block = nil
		bS.isProcessed = true
		m.blockStatuses[f.Metadata.BlockID] = bS

		return sourceSymbolBytes, nil
	}
	// the block is still not recoverable, so we wait
	m.blockStatuses[f.Metadata.BlockID] = bS
	return nil, nil
}

func (m *manager) HandleSourceSymbolFrame(f *wire.SourceSymbolFrame) ([]byte, error) {
	blockID := m.sidToBlockID(f.SSID)
	if _, exists := m.blockStatuses[blockID]; !exists {
		// create a new block if it doesn't exist
		m.blockStatuses[blockID] = blockStatus{
			block:       newBlock(blockID, m.numTotSourceSymbols, m.numTotRepairSymbols),
			isProcessed: false,
		}
	}

	bS := m.blockStatuses[blockID]
	if bS.isProcessed {
		// we've already processed the block, so we can ignore this source symbol
		return nil, nil
	}

	err := bS.block.addSourceSymbol(f)
	if err != nil {
		return nil, err
	}

	if bS.block.isComplete() {
		bS.block = nil
		bS.isProcessed = true
	}
	m.blockStatuses[blockID] = bS
	return f.Payload, nil
}

// TODO (ddritzenhoff) repair symbols are always created here, which should make it possible to allocate a set of repair symbols using sync.pool and always fetch new ones from there.
