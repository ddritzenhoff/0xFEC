package fec

import "github.com/quic-go/quic-go/internal/wire"

type BlockFECScheme interface {
	// repairSymbols generates repair symbols for the block. An error is returned if the block is not complete.
	repairSymbols(b *block) ([]*wire.RepairFrame, error)
	// recoverSymbols reconstructs the missing source symbols of the block and returns them as a slice. An error is returned if there aren't enough present symbols to repair the missing ones.
	recoverSymbolPayloads(b *block) ([]byte, error)
}
