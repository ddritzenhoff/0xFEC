package fec

import (
	"fmt"

	"github.com/klauspost/reedsolomon"
	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/wire"
)

type reedSolomonScheme struct {
	enc reedsolomon.Encoder
}

func NewReedSolomonScheme(numTotSourceSymbols int, numTotRepairSymbols int) (*reedSolomonScheme, error) {
	enc, err := reedsolomon.New(numTotSourceSymbols, numTotRepairSymbols)
	if err != nil {
		return nil, err
	}
	return &reedSolomonScheme{
		enc: enc,
	}, nil
}

// repairSymbols generates repair symbols for the block. An error is returned if the block is not full with source symbols.
func (s *reedSolomonScheme) repairSymbols(b *block) ([]*wire.RepairFrame, error) {
	if !b.isComplete() {
		return nil, fmt.Errorf("block does not have enough source symbols to generate repair symbols")
	}

	if b.biggestSourceSymbolLenSoFar > protocol.MaxFECPacketBufferSize {
		return nil, fmt.Errorf("source symbol payload len is greater is too big for FEC headers. Max %d and got %d", protocol.MaxFECPacketBufferSize, b.biggestSourceSymbolLenSoFar)
	}

	shards := make([][]byte, b.totNumSourceSymbols+b.totNumRepairSymbols)
	for i := 0; i < b.totNumSourceSymbols; i++ {
		shardPayload, err := s.addLengthToSourceSymbolPayload(b, b.smallestSSID+protocol.SourceSymbolID(i))
		if err != nil {
			return nil, err
		}
		shards[i] = shardPayload
	}

	for i := 0; i < b.totNumRepairSymbols; i++ {
		// TODO: you may not need the protocol.MaxPacketBufferSize capacity on the sending side (which this is).
		repairShard := make([]byte, 0, protocol.MaxPacketBufferSize)
		repairShard = repairShard[:protocol.RepairPayloadMetadataLen+b.biggestSourceSymbolLenSoFar]
		shards[i+b.totNumSourceSymbols] = repairShard
	}

	err := s.enc.Encode(shards)
	if err != nil {
		return nil, fmt.Errorf("unable to make parity shards: %w", err)
	}

	repairSymbols := make([]*wire.RepairFrame, b.totNumRepairSymbols)
	for i := range repairSymbols {
		repairSymbols[i] = &wire.RepairFrame{
			Metadata: protocol.BlockMetadata{
				BlockID:  b.id,
				ParityID: protocol.ParityID(i),
			},
			Payload: shards[b.totNumSourceSymbols+i],
		}
	}

	return repairSymbols, nil
}

func (s *reedSolomonScheme) addLengthToSourceSymbolPayload(b *block, ssid protocol.SourceSymbolID) ([]byte, error) {
	payload, exists := b.ssidToSourcePayload[ssid]
	if !exists {
		// this should never happen
		return nil, fmt.Errorf("block [%d, %d] is complete but SID %d does not exist", b.smallestSSID, b.largestSSID, ssid)
	}

	payloadLen := uint16(len(payload))
	highByte := byte(payloadLen >> 8)  // High byte
	lowByte := byte(payloadLen & 0xFF) // Low byte
	shardLen := protocol.RepairPayloadMetadataLen + b.biggestSourceSymbolLenSoFar
	if shardLen > cap(payload) {
		// this should never happen
		return nil, fmt.Errorf("shard len (%d) is greater than capacity of payload (%d)", shardLen, cap(payload))
	}
	shardPayload := payload[:shardLen]
	shardPayload[b.biggestSourceSymbolLenSoFar] = highByte
	shardPayload[b.biggestSourceSymbolLenSoFar+1] = lowByte
	return shardPayload, nil
}

// recoverSymbols reconstructs the missing source symbols of the block and returns them as a slice. An error is returned if there aren't enough present symbols to repair the missing ones.
func (s *reedSolomonScheme) recoverSymbolPayloads(b *block) ([]byte, error) {
	if !b.isRecoverable() {
		return nil, fmt.Errorf("not enough present symbols to repair the missing ones")
	}

	if b.isComplete() {
		// The block is complete, so there's nothing to be recovered
		return nil, nil
	}

	shards := make([][]byte, b.totNumSourceSymbols+b.totNumRepairSymbols)
	numMissingSourceSymbols := b.totNumSourceSymbols - len(b.ssidToSourcePayload)
	missingSourceShardIndices := make([]int, 0, numMissingSourceSymbols)
	for i := 0; i < b.totNumSourceSymbols; i++ {
		ssid := b.smallestSSID + protocol.SourceSymbolID(i)
		if _, exists := b.ssidToSourcePayload[ssid]; !exists {
			missingSourceShardIndices = append(missingSourceShardIndices, i)
			shards[i] = nil
		} else {
			shardPayload, err := s.addLengthToSourceSymbolPayload(b, ssid)
			if err != nil {
				return nil, err
			}
			shards[i] = shardPayload
		}
	}

	for parityID, payload := range b.pidToRepairPayload {
		i := b.totNumSourceSymbols + int(parityID)
		shards[i] = payload
	}

	if err := s.enc.ReconstructData(shards); err != nil {
		return nil, err
	}

	recoveredSymbolPayloads := make([]byte, 0, numMissingSourceSymbols*b.biggestSourceSymbolLenSoFar)
	for _, i := range missingSourceShardIndices {
		missingSourceShard := shards[i]
		payloadLen := uint16(missingSourceShard[b.biggestSourceSymbolLenSoFar])<<8 | uint16(missingSourceShard[b.biggestSourceSymbolLenSoFar+1])
		recoveredSymbolPayloads = append(recoveredSymbolPayloads, missingSourceShard[:payloadLen]...)
	}

	return recoveredSymbolPayloads, nil
}
