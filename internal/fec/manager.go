package fec

import (
	"fmt"
	"sync"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/wire"
)

type Manager interface {
	HandleRepairFrame(f *wire.RepairFrame) ([]byte, error)
	// HandleSourceSymbolFrame is a receiver-side function
	HandleSourceSymbolFrame(f *wire.SourceSymbolFrame) ([]byte, error)
	// HandleSymbolAckFrame()
	// HandleFECWindowFrame()
	NextSID() protocol.SID
	UpdateWindowSize(newSize protocol.FECWindowSize, epoch protocol.FECWindowEpoch) error
	// AddSourceSymbolFrame is a sender-side function
	AddSourceSymbolFrame(f *wire.SourceSymbolFrame) ([]*wire.RepairFrame, error)
	SetInitialCodingWindow(ws protocol.FECWindowSize)
}

type manager struct {
	nextSIDMutex sync.Mutex
	nextSID      protocol.SID
	params       blockParams
	window       fecWindow
	blocks       map[protocol.BlockID]*Block
	// TODO (ddritzenhoff) come up with a better way to do this. Could I do something like having an integer keep track of the lowest blockID block to already have been processed?
	// NOTE: this is a receiver-side only data structure.
	processedBlocks map[protocol.BlockID]struct{}
}

func NewFECManager() *manager {
	return &manager{
		// TODO (ddritzenhoff) Change the FEC scheme to be dynamic.
		params: blockParams{
			n: 3,
			k: 2,
		},
		window:          *newFECWindow(),
		blocks:          make(map[protocol.BlockID]*Block),
		processedBlocks: make(map[protocol.BlockID]struct{}),
		nextSID:         0,
	}
}

func (m *manager) NextSID() protocol.SID {
	m.nextSIDMutex.Lock()
	ret := m.nextSID
	m.nextSID++
	m.nextSIDMutex.Unlock()
	return ret
}

func (m *manager) sidToBlockID(sid protocol.SID) protocol.BlockID {
	return protocol.BlockID(uint64(sid) / uint64(m.params.k))
}

func (m *manager) AddSourceSymbolFrame(f *wire.SourceSymbolFrame) ([]*wire.RepairFrame, error) {
	blockID := m.sidToBlockID(f.SID)
	block, blockExists := m.blocks[blockID]
	if !blockExists {
		block = newBlock(blockID, m.params)
		m.blocks[blockID] = block
	}

	isBlockFull := block.addSourceSymbol(f)
	if isBlockFull {
		repairFrames, err := block.repairFrames()
		if err != nil {
			return nil, err
		}
		// drop the block as you don't need it anymore.
		delete(m.blocks, blockID)
		return repairFrames, nil
	}
	return nil, nil
}

func (m *manager) HandleRepairFrame(f *wire.RepairFrame) ([]byte, error) {
	if m.sidToBlockID(f.RID.SmallestSID) != m.sidToBlockID(f.RID.LargestSID) {
		return nil, fmt.Errorf("RID smallestSID (%d) and largestSID (%d) correspond to different blocks", f.RID.SmallestSID, f.RID.LargestSID)
	}
	blockID := m.sidToBlockID(f.RID.SmallestSID)
	if _, processedBlock := m.processedBlocks[blockID]; processedBlock {
		// we've already processed the block, so we can ignore this repair symbol.
		return nil, nil
	}
	block, blockExists := m.blocks[blockID]
	if !blockExists {
		return nil, fmt.Errorf("repair symbol generated for block that doesn't exist and hasn't been processed")
	}
	isBlockFull := block.addRepairSymbol(f)
	if isBlockFull {
		blockData, err := block.data()
		if err != nil {
			return nil, err
		}
		m.processedBlocks[blockID] = struct{}{}
		delete(m.blocks, blockID)
		return blockData, nil
	}
	return nil, nil
}

// TODO (ddritzenhoff) need to add in FEC_WINDOW logic. Right now, I would store every single symbol.
func (m *manager) HandleSourceSymbolFrame(f *wire.SourceSymbolFrame) ([]byte, error) {
	blockID := m.sidToBlockID(f.SID)
	if _, processedBlock := m.processedBlocks[blockID]; processedBlock {
		// we've already processed the block, so we can ignore this source symbol.
		return nil, nil
	}
	block, blockExists := m.blocks[blockID]
	if !blockExists {
		block = newBlock(blockID, m.params)
		m.blocks[blockID] = block
	}
	isBlockFull := block.addSourceSymbol(f)
	if isBlockFull {
		// Only retrieve the data that hasn't already been passed to the application.
		blockData, err := block.data()
		if err != nil {
			return nil, err
		}
		// We have collected all of the requisite source symbols for the respective block, so we can now drop the block entirely.
		m.processedBlocks[blockID] = struct{}{}
		delete(m.blocks, blockID)
		return blockData, nil
	}
	// Mark this source symbol as already having been passed up to the application, which ensures it'll only be passed once.
	return block.processSymbol(f)
}

// UpdateWindowSize updates the window size, which denotes the maximum number of received source symbols that can be stored at a time.
func (m *manager) UpdateWindowSize(newWindowSize protocol.FECWindowSize, epoch protocol.FECWindowEpoch) error {
	return m.window.update(newWindowSize, epoch)
}

func (m *manager) SetInitialCodingWindow(ws protocol.FECWindowSize) {
	m.window.setInitialCodingWindow(ws)
}

type blockParams struct {
	// n denotes the total number of symbols, including repair symbols, for the block
	n int
	// k denotes the total number of source symbols in the block
	k int
	// (n-k) indicates the number of repair symbols within the scheme (e.g. (3, 2) denotes 2 source symbols and (3-2=1) repair symbols for a total of 3 symbols in the block)
}

type fecWindow struct {
	size       protocol.FECWindowSize
	epoch      protocol.FECWindowEpoch
	hasBeenSet bool
}

func newFECWindow() *fecWindow {
	return &fecWindow{
		size:       0,
		epoch:      0,
		hasBeenSet: false,
	}
}

func (fw *fecWindow) update(newWindowSize protocol.FECWindowSize, epoch protocol.FECWindowEpoch) error {
	if epoch == 0 && fw.epoch == 0 && !fw.hasBeenSet {
		fw.size = newWindowSize
		fw.hasBeenSet = true
	} else if epoch > fw.epoch {
		fw.epoch = epoch
		fw.size = newWindowSize
	} else {
		// TODO (ddritzenhoff) it could be that we should just ignore invalid FEC_WINDOW updates. This is ambiguous within the spec, but it could definitely be something to ask about.
		return fmt.Errorf("invalid fec window update: newWindowSize %d, newEpoch %d, windowSize: %d, epoch %d", newWindowSize, epoch, fw.size, fw.epoch)
	}
	return nil
}

func (fw *fecWindow) setInitialCodingWindow(ws protocol.FECWindowSize) {
	fw.size = ws
}
