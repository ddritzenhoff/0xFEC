package protocol

type FECSchemeID byte

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
