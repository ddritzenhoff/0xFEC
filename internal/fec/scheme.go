package fec

import "github.com/quic-go/quic-go/internal/wire"

type BlockScheme interface {
	GetRepairSymbols(block *Block, numberOfSymbols uint) ([]*wire.RepairFrame, error)
	// RecoverSymbols recovers the missing source symbols using the existing repair and source symbols. On success, nil is returned.
	// TODO (ddritzenhoff) Maybe change the function signature for extra convenience.
	RecoverSymbols(block *Block) error
}
