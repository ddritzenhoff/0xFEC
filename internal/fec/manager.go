package fec

import (
	"fmt"
	"sort"
	"sync"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/wire"
)

type Manager interface {
	// HandleRepairFrame()
	HandleSourceSymbolFrame(f *wire.SourceSymbolFrame) (isBlockFull bool, blockData []byte, _ error)
	// HandleSymbolAckFrame()
	// HandleFECWindowFrame()
	NextSID() protocol.SID
	UpdateWindowSize(newSize protocol.FECWindowSize, epoch protocol.FECWindowEpoch) error
	SetInitialCodingWindow(ws protocol.FECWindowSize)
}

type manager struct {
	nextSIDMutex sync.Mutex
	nextSID      protocol.SID
	scheme       fecScheme
	window       fecWindow
	blocks       map[protocol.BlockID]*Block
}

func NewFECManager() *manager {
	return &manager{
		// TODO (ddritzenhoff) Change the FEC scheme to be dynamic.
		scheme: fecScheme{
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
	return protocol.BlockID(uint64(sid) / uint64(m.scheme.k))
}

// TODO (ddritzenhoff) need to add in FEC_WINDOW logic. Right now, I would store every single symbol.
func (m *manager) HandleSourceSymbolFrame(f *wire.SourceSymbolFrame) (isBlockFull bool, blockData []byte, _ error) {
	blockID := m.sidToBlockID(f.SID)
	block, ok := m.blocks[blockID]
	if !ok {
		block = newBlock(blockID, m.scheme)
		m.blocks[blockID] = block
	}

	isBlockFull = block.addSourceSymbol(f)
	if isBlockFull {
		blockData, err := block.data()
		if err != nil {
			return true, nil, err
		}
		return true, blockData, nil
	}
	return false, nil, nil
}

// UpdateWindowSize updates the window size, which denotes the maximum number of received source symbols that can be stored at a time.
func (m *manager) UpdateWindowSize(newWindowSize protocol.FECWindowSize, epoch protocol.FECWindowEpoch) error {
	return m.window.update(newWindowSize, epoch)
}

func (m *manager) SetInitialCodingWindow(ws protocol.FECWindowSize) {
	m.window.setInitialCodingWindow(ws)
}

type fecScheme struct {
	// n denotes the total number of symbols, including repair symbols, for the block
	n uint8
	// k denotes the total number of source symbols in the block
	k uint8
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

/*
TODO
- finish deciding how transport parameters will be sent back and forth between the two peers: applyTransportParameters() is probably the right function here.
- figure out how to handle the source symbol frame. Adding an addition within handleFrames() is probably the right move here.
*/

type Block struct {
	id                    protocol.BlockID
	scheme                fecScheme
	sidToSourceData       map[protocol.SID][]byte
	biggestSourceLenSoFar int
	ridToRepairData       map[protocol.RID][]byte
	nextRID               protocol.RID
}

func newBlock(id protocol.BlockID, scheme fecScheme) *Block {
	return &Block{
		id:                    id,
		scheme:                scheme,
		sidToSourceData:       make(map[protocol.SID][]byte),
		biggestSourceLenSoFar: 0,
		ridToRepairData:       make(map[protocol.RID][]byte),
		nextRID:               0,
	}
}

func (b *Block) data() ([]byte, error) {
	if len(b.sidToSourceData) == int(b.scheme.k) {
		// we have all the needed source symbols, so we can just combine them.
		return b.combineSourceSymbols(), nil
	} else if len(b.sidToSourceData)+len(b.ridToRepairData) >= int(b.scheme.k) {
		// TODO (ddritzenhoff) finish this implementation.
		return nil, nil
	}
	return nil, fmt.Errorf("fec block is not full")
}

func (b *Block) addRepairSymbol(f *wire.RepairFrame) (isBlockFull bool) {
	if _, ok := b.ridToRepairData[f.RID]; !ok {
		b.ridToRepairData[f.RID] = f.Data
	}
	return len(b.sidToSourceData)+len(b.ridToRepairData) >= int(b.scheme.k) || len(b.sidToSourceData) == int(b.scheme.k)
}

func (b *Block) addSourceSymbol(f *wire.SourceSymbolFrame) (isBlockFull bool) {
	if _, ok := b.sidToSourceData[f.SID]; !ok {
		b.sidToSourceData[f.SID] = f.Payload
		if b.biggestSourceLenSoFar < len(f.Payload) {
			b.biggestSourceLenSoFar = len(f.Payload)
		}
	}
	// The block is full when there are enough source symbols to satisfy k in the (n, k) block encoding scheme. Alternatively, the block is also full when the number of source symbols + the number of repair symbols are equal or greater than k.
	return len(b.sidToSourceData) == int(b.scheme.k) || len(b.sidToSourceData)+len(b.ridToRepairData) >= int(b.scheme.k)
}

func (b *Block) combineSourceSymbols() []byte {
	// Get and sort the keys.
	sids := make([]protocol.SID, 0, len(b.sidToSourceData))
	for sid := range b.sidToSourceData {
		sids = append(sids, sid)
	}
	sort.Slice(sids, func(i, j int) bool { return sids[i] < sids[j] })

	// Combine the source symbols in order.
	var combined []byte
	for _, sid := range sids {
		combined = append(combined, b.sidToSourceData[sid]...)
	}

	return combined
}

func (b *Block) generateRID() protocol.RID {
	ret := b.nextRID
	b.nextRID++
	return ret
}
