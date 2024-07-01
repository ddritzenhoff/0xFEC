package protocol

import "fmt"

type FECSchemeID byte

type FECWindowEpoch uint16
type FECWindowSize uint32

type SID uint64

// TODO (ddritzenhoff) This is just a way to distinguish the repair frames that support the same block. This may prove to be unnecessary, so just be sure to double check this.
type RID struct {
	SmallestSID SID
	LargestSID  SID
}

func NewRID(smallestSID SID, largestSID SID) (RID, error) {
	if smallestSID >= largestSID {
		return RID{}, fmt.Errorf("SmallestSID must be smaller than LargestSID: SmallestSID %d, LargestSID %d", smallestSID, largestSID)
	}

	return RID{
		smallestSID,
		largestSID,
	}, nil
}

type BlockID uint64

const (
	FECDisabled          FECSchemeID = iota // 0x0
	XORFECScheme                            // 0x1
	ReedSolomonFECScheme                    // 0x2
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
