package protocol

type FECWindowEpoch uint16
type FECWindowSize uint32

// SourceSymbolID (aka. SSID) represents the ID for a source symbol. These are unique.
type SourceSymbolID uint64

// BlockMetadata represents the requisite metadata for block encoding schemes.
type BlockMetadata struct {
	BlockID  BlockID
	ParityID ParityID
}

// BlockID represents the ID of the block. IDs of blocks start at 0 and increase by 1 for each subsequent block.
type BlockID uint64

// ParityID represents the order of parity symbols within the block. This is important to properly reconstruct missing source symbols.
type ParityID uint64

type DecoderFECScheme byte

const (
	FECDisabled          DecoderFECScheme = iota // 0x0
	XORFECScheme                                 // 0x1
	ReedSolomonFECScheme                         // 0x2
)

func (f DecoderFECScheme) String() string {
	switch f {
	case XORFECScheme:
		return "XOR"
	case ReedSolomonFECScheme:
		return "ReedSolomon"
	default:
		return "unknown"
	}
}
