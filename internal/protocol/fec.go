package protocol

type FECSchemeID byte

type FECWindowEpoch uint16
type FECWindowSize uint32

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
