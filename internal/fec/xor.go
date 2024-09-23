package fec

import (
	"fmt"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/wire"
)

type xorScheme struct {
}

// repairSymbols generates repair symbols for the block. An error is returned if the block is not full with source symbols.
func (s *xorScheme) repairSymbols(b *block) ([]*wire.RepairFrame, error) {
	if !b.isComplete() {
		return nil, fmt.Errorf("block does not have enough source symbols to generate repair symbols")
	}

	if b.totNumRepairSymbols != 1 {
		return nil, fmt.Errorf("xor only supports 1 repair symbol. Expected 1, received %d", b.totNumRepairSymbols)
	}

	if b.biggestSourceSymbolLenSoFar > protocol.MaxFECPacketBufferSize {
		return nil, fmt.Errorf("source symbol payload len is greater is too big for FEC headers. Max %d and got %d", protocol.MaxFECPacketBufferSize, b.biggestSourceSymbolLenSoFar)
	}

	// Need 2 additional bytes to encode the length of the source symbol.
	repairPayloadLen := protocol.RepairPayloadMetadataLen + b.biggestSourceSymbolLenSoFar
	xorSoFar := make([]byte, repairPayloadLen)
	for _, payload := range b.ssidToSourcePayload {
		xorSoFar = s.xor(xorSoFar, payload, b.biggestSourceSymbolLenSoFar)
	}

	return []*wire.RepairFrame{
		{
			Metadata: protocol.BlockMetadata{
				BlockID:  b.id,
				ParityID: 0,
			},
			Payload: xorSoFar,
		}}, nil
}

func (s *xorScheme) xor(xorSoFar []byte, payload []byte, biggestSourceSymbolLenSoFar int) []byte {
	for i := 0; i < len(payload); i++ {
		xorSoFar[i] = xorSoFar[i] ^ payload[i]
	}
	// XORing the length of the payload to xorSoFar, so the correct length can later be extracted after repairing.
	payloadLen := uint16(len(payload))
	highByte := byte(payloadLen >> 8)  // High byte
	lowByte := byte(payloadLen & 0xFF) // Low byte
	xorSoFar[biggestSourceSymbolLenSoFar] ^= highByte
	xorSoFar[biggestSourceSymbolLenSoFar+1] ^= lowByte

	return xorSoFar
}

func (s *xorScheme) xorRepair(xorSoFar []byte, payload []byte) []byte {
	for i := 0; i < len(payload); i++ {
		xorSoFar[i] = xorSoFar[i] ^ payload[i]
	}
	return xorSoFar
}

// recoverSymbols reconstructs the missing source symbols of the block and returns them as a slice. An error is returned if there aren't enough present symbols to repair the missing ones.
func (s *xorScheme) recoverSymbolPayloads(b *block) ([]byte, error) {
	if !b.isRecoverable() {
		return nil, fmt.Errorf("not enough present symbols to repair the missing ones")
	}

	if b.isComplete() {
		// The block is complete, so there's nothing to be recovered
		return nil, nil
	}

	// the repair symbol will have the same size as the biggest source symbol.

	recoveredSymbol := make([]byte, protocol.MaxPacketBufferSize)
	for _, data := range b.pidToRepairPayload {
		recoveredSymbol = s.xorRepair(recoveredSymbol, data)
	}
	for _, payload := range b.ssidToSourcePayload {
		recoveredSymbol = s.xor(recoveredSymbol, payload, b.biggestSourceSymbolLenSoFar)
	}

	// at this point, the symbol should be recovered. We just have to trim the extra zeros that may be hanging at the end. The first two bytes of the recovered symbol indicate the length.

	payloadLen := uint16(recoveredSymbol[b.biggestSourceSymbolLenSoFar])<<8 | uint16(recoveredSymbol[b.biggestSourceSymbolLenSoFar+1])
	recoveredPayload := recoveredSymbol[:payloadLen]

	for ssid := b.smallestSSID; ssid <= b.largestSSID; ssid++ {
		// because XOR can only handle one loss, we can stop at the first missing SSID.
		if _, exists := b.ssidToSourcePayload[ssid]; !exists {
			b.ssidToSourcePayload[ssid] = recoveredPayload
		}
	}

	if !b.isComplete() {
		// this is a sanity check, which should never happen
		return nil, fmt.Errorf("block is not complete after recovery")
	}

	return recoveredPayload, nil
}
