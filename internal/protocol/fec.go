package protocol

type FECSchemeID byte

type FECWindowEpoch uint16
type FECWindowSize uint32

// TODO (ddritzenhoff) should probably rename this. It used to be FECSymbolID, but that was too close to FECSchemeID
type SID uint32

// This is just a way to distinguish the repair frames that support the same block. This may prove to be unnecessary. I'm not sure yet.
type RID uint32

type BlockID uint32

// TODO (ddritzenhoff) is it better to use an enum here?
const FECDisabled FECSchemeID = 0
const XORFECScheme FECSchemeID = 1
const ReedSolomonFECScheme FECSchemeID = 2

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
