package fec

import (
	"fmt"
	"sync"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/wire"
)

type Manager interface {
	// HandleRepairFrame()
	// HandleSourceSymbolFrame is a receiver-side function
	HandleSourceSymbolFrame(f *wire.SourceSymbolFrame) (blockData []byte, _ error)
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
}

func NewFECManager() *manager {
	return &manager{
		// TODO (ddritzenhoff) Change the FEC scheme to be dynamic.
		params: blockParams{
			n: 3,
			k: 2,
		},
		window:  *newFECWindow(),
		blocks:  make(map[protocol.BlockID]*Block),
		nextSID: 0,
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
	block, ok := m.blocks[blockID]
	if !ok {
		block = newBlock(blockID, m.params)
		m.blocks[blockID] = block
	}

	isBlockFull := block.addSourceSymbol(f)
	if isBlockFull {
		repairFrames, err := block.repairFrames()
		if err != nil {
			return nil, err
		}
		// TODO (ddritzenhoff) keep an eye on this and make sure it's not a mistake.
		// drop the block as you don't need it anymore.
		delete(m.blocks, blockID)
		return repairFrames, nil
	}
	return nil, nil
}

// TODO (ddritzenhoff) need to add in FEC_WINDOW logic. Right now, I would store every single symbol.
func (m *manager) HandleSourceSymbolFrame(f *wire.SourceSymbolFrame) (blockData []byte, _ error) {
	blockID := m.sidToBlockID(f.SID)
	block, ok := m.blocks[blockID]
	if !ok {
		block = newBlock(blockID, m.params)
		m.blocks[blockID] = block
	}

	isBlockFull := block.addSourceSymbol(f)
	if isBlockFull {
		blockData, err := block.data()
		if err != nil {
			return nil, err
		}
		// We have collected all of the requisite source symbols for the respective block, so we can now drop the block entirely.
		delete(m.blocks, blockID)
		return blockData, nil
	}
	return nil, nil
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
	// (n-k) will then provide you with the number of repair symbols within the scheme (e.g. (3, 2) denotes 2 source symbols and (3-2) repair symbols for a total of 3 symbols in the block)
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
