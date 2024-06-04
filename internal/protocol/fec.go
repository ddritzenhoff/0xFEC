package protocol

type FECSchemeID byte

type FECWindowEpoch uint16
type FECWindowSize uint32

type SID uint64

// TODO (ddritzenhoff) This is just a way to distinguish the repair frames that support the same block. This may prove to be unnecessary, so just be sure to double check this.
type RID uint64

type BlockID uint64

const (
	FECDisabled FECSchemeID = iota
	XORFECScheme
	ReedSolomonFECScheme
)

func (f FECSchemeID) String() string {
	switch f {
	case XORFECScheme:
		return "XOR"
	case ReedSolomonFECScheme:
		return "ReedSolomon"
	default:
		return "unknown"
	}
}
