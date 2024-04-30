package wire

import (
	"bytes"
	"errors"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/quicvarint"
)

// TODO (ddritzenhoff) I don't understand the ACK ranges. If there is a gap, will that gap always be filled?
// TODO (ddritzenhoff) may have to do more work in internal/ackhandler/(sent_packet_handler.go)

/*
It seems as if the symbol ack frame is specifically used to denote the IDs of the recovered source symbols.
*/

var errInvalidSymbolAckRanges = errors.New("SymbolAckFrame: Symbol ACK frame contains invalid ACK ranges")

type SymbolAckFrame struct {
	AckRanges []AckRange
}

func parseSymbolAckFrame(frame *SymbolAckFrame, r *bytes.Reader, _ protocol.Version) error {
	la, err := quicvarint.Read(r)
	if err != nil {
		return err
	}
	largestAcked := protocol.PacketNumber(la)
	numBlocks, err := quicvarint.Read(r)
	if err != nil {
		return err
	}
	// read the first ACK range
	ab, err := quicvarint.Read(r)
	if err != nil {
		return err
	}
	ackBlock := protocol.PacketNumber(ab)
	if ackBlock > largestAcked {
		return errors.New("invalid first ACK range")
	}
	smallest := largestAcked - ackBlock
	frame.AckRanges = append(frame.AckRanges, AckRange{Smallest: smallest, Largest: largestAcked})

	// read all the other ACK ranges
	for i := uint64(0); i < numBlocks; i++ {
		g, err := quicvarint.Read(r)
		if err != nil {
			return err
		}
		gap := protocol.PacketNumber(g)
		if smallest < gap+2 {
			return errInvalidSymbolAckRanges
		}
		largest := smallest - gap - 2

		ab, err := quicvarint.Read(r)
		if err != nil {
			return err
		}
		ackBlock := protocol.PacketNumber(ab)

		if ackBlock > largest {
			return errInvalidSymbolAckRanges
		}
		smallest = largest - ackBlock
		frame.AckRanges = append(frame.AckRanges, AckRange{Smallest: smallest, Largest: largest})
	}

	if !frame.validateAckRanges() {
		return errInvalidSymbolAckRanges
	}

	return nil
}

func (f *SymbolAckFrame) validateAckRanges() bool {
	if len(f.AckRanges) == 0 {
		return false
	}

	// check the validity of every single ACK range
	for _, ackRange := range f.AckRanges {
		if ackRange.Smallest > ackRange.Largest {
			return false
		}
	}

	// check the consistency for ACK with multiple NACK ranges
	for i, ackRange := range f.AckRanges {
		if i == 0 {
			continue
		}
		lastAckRange := f.AckRanges[i-1]
		if lastAckRange.Smallest <= ackRange.Smallest {
			return false
		}
		if lastAckRange.Smallest <= ackRange.Largest+1 {
			return false
		}
	}

	return true
}

// Append appends an ACK frame.
func (f *SymbolAckFrame) Append(b []byte, _ protocol.Version) ([]byte, error) {
	b = quicvarint.Append(b, uint64(f.LargestAcked()))
	numRanges := f.numEncodableAckRanges()
	b = quicvarint.Append(b, uint64(numRanges-1))
	// write the first range
	_, firstRange := f.encodeAckRange(0)
	b = quicvarint.Append(b, firstRange)
	// write all the other range
	for i := 1; i < numRanges; i++ {
		gap, len := f.encodeAckRange(i)
		b = quicvarint.Append(b, gap)
		b = quicvarint.Append(b, len)
	}
	return b, nil
}

// LargestAcked is the largest acked packet number
func (f *SymbolAckFrame) LargestAcked() protocol.PacketNumber {
	return f.AckRanges[0].Largest
}

// gets the number of ACK ranges that can be encoded
// such that the resulting frame is smaller than the maximum ACK frame size
func (f *SymbolAckFrame) numEncodableAckRanges() int {
	// TODO (ddritzenhoff) what's the deal with this 1? I assume it's for the byte it takes to encode the Type, but I'm not positive.
	// length := 1 + quicvarint.Len(uint64(f.LargestAcked()))
	length := quicvarint.Len(uint64(f.LargestAcked()))
	length += 2 // assume that the number of ranges will consume 2 bytes
	for i := 1; i < len(f.AckRanges); i++ {
		gap, len := f.encodeAckRange(i)
		rangeLen := quicvarint.Len(gap) + quicvarint.Len(len)
		if length+rangeLen > protocol.MaxAckFrameSize {
			// Writing range i would exceed the MaxAckFrameSize.
			// So encode one range less than that.
			return i - 1
		}
		length += rangeLen
	}
	return len(f.AckRanges)
}

func (f *SymbolAckFrame) encodeAckRange(i int) (uint64 /* gap */, uint64 /* length */) {
	if i == 0 {
		return 0, uint64(f.AckRanges[0].Largest - f.AckRanges[0].Smallest)
	}
	return uint64(f.AckRanges[i-1].Smallest - f.AckRanges[i].Largest - 2),
		uint64(f.AckRanges[i].Largest - f.AckRanges[i].Smallest)
}

// Length of a written frame
func (f *SymbolAckFrame) Length(_ protocol.Version) protocol.ByteCount {
	largestAcked := f.AckRanges[0].Largest
	numRanges := f.numEncodableAckRanges()

	length := quicvarint.Len(uint64(largestAcked))

	length += quicvarint.Len(uint64(numRanges - 1))
	lowestInFirstRange := f.AckRanges[0].Smallest
	length += quicvarint.Len(uint64(largestAcked - lowestInFirstRange))

	for i := 1; i < numRanges; i++ {
		gap, len := f.encodeAckRange(i)
		length += quicvarint.Len(gap)
		length += quicvarint.Len(len)
	}
	return length
}

func (f *SymbolAckFrame) Reset() {
	for _, r := range f.AckRanges {
		r.Largest = 0
		r.Smallest = 0
	}
	// Doesn't change the capacity (keeps the underlying array), but sets the length back down to 0.
	f.AckRanges = f.AckRanges[:0]
}
