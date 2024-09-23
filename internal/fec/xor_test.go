package fec

import (
	"reflect"
	"testing"

	"github.com/quic-go/quic-go/internal/protocol"
	"github.com/quic-go/quic-go/internal/wire"
)

func TestXorScheme_RepairSymbols(t *testing.T) {
	scheme := &xorScheme{}

	tests := []struct {
		name    string
		block   *block
		want    []*wire.RepairFrame
		wantErr bool
	}{
		{
			name: "complete block with correct number of repair symbols",
			block: &block{
				id:                          1,
				totNumRepairSymbols:         1,
				totNumSourceSymbols:         2,
				biggestSourceSymbolLenSoFar: 6,
				ssidToSourcePayload: map[protocol.SourceSymbolID][]byte{
					1: {1, 2, 3, 3, 2, 7},
					2: {4, 3, 2, 1},
				},
			},
			want: []*wire.RepairFrame{
				{
					Metadata: protocol.BlockMetadata{
						BlockID:  1,
						ParityID: 0,
					},
					/*
						total len of the repair payload should be b.biggestSourceSymbolLenSoFar + 2
					*/
					Payload: []byte{5, 1, 1, 2, 2, 7, 0, 2},
				},
			},
			wantErr: false,
		},
		{
			name: "block not complete",
			block: &block{
				ssidToSourcePayload: map[protocol.SourceSymbolID][]byte{
					1: {0, 1, 2},
				},
				totNumSourceSymbols: 2,
				totNumRepairSymbols: 1,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "more than one repair symbol",
			block: &block{
				ssidToSourcePayload: map[protocol.SourceSymbolID][]byte{
					1: {0, 1, 2},
					2: {3, 4, 5},
				},
				totNumSourceSymbols: 2,
				totNumRepairSymbols: 2,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "source symbol length too big",
			block: &block{
				ssidToSourcePayload: map[protocol.SourceSymbolID][]byte{
					1: make([]byte, protocol.MaxFECPacketBufferSize+1),
					2: make([]byte, protocol.MaxFECPacketBufferSize),
				},
				totNumSourceSymbols:         2,
				totNumRepairSymbols:         1,
				biggestSourceSymbolLenSoFar: protocol.MaxFECPacketBufferSize + 1,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "successful repair symbol generation with large different-sized slices",
			block: &block{
				id: 0,
				ssidToSourcePayload: map[protocol.SourceSymbolID][]byte{
					1: {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
					2: {16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30},
				},
				totNumSourceSymbols:         2,
				totNumRepairSymbols:         1,
				biggestSourceSymbolLenSoFar: 16,
			},
			want: []*wire.RepairFrame{
				{
					Metadata: protocol.BlockMetadata{
						BlockID:  0,
						ParityID: 0,
					},
					Payload: []byte{16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 15, 0, 31},
				},
			},
			wantErr: false,
		},
		{
			name: "successful repair symbol generation with slices of sizes 1434, 1315, and 1400",
			block: &block{
				id: 0,
				ssidToSourcePayload: map[protocol.SourceSymbolID][]byte{
					1: generateLargePayload(1434, 0x1),
					2: generateLargePayload(1315, 0x2),
					3: generateLargePayload(1400, 0x4),
				},
				totNumSourceSymbols:         3,
				totNumRepairSymbols:         1,
				biggestSourceSymbolLenSoFar: 1434,
			},
			want: []*wire.RepairFrame{
				{
					Metadata: protocol.BlockMetadata{
						BlockID:  0,
						ParityID: 0,
					},
					Payload: func() []byte {
						payload := make([]byte, 1436) // 1434 + 2 bytes for length encoding
						payload = generateExpectedXORPayload(map[protocol.SourceSymbolID][]byte{
							1: generateLargePayload(1434, 0x1),
							2: generateLargePayload(1315, 0x2),
							3: generateLargePayload(1400, 0x4),
						}, len(payload))
						xorLen := uint16(1434 ^ 1315 ^ 1400)

						payload[1434] = byte(xorLen >> 8)
						payload[1435] = byte(xorLen & 0xFF)
						return payload
					}(),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scheme.repairSymbols(tt.block)
			if (err != nil) != tt.wantErr {
				t.Errorf("repairSymbols() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.want) != len(got) {
				t.Errorf("repairSymbols() error = slices not the same length")
				return
			}
			for i := 0; i < len(got); i++ {
				if !reflect.DeepEqual(got[i], tt.want[i]) {
					t.Errorf("repairSymbols() got %v, want %v", got[i], tt.want[i])
				}
			}
		})
	}
}

// generateLargePayload generates a byte slice of the given size filled with the specified value.
func generateLargePayload(size int, value byte) []byte {
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = value
	}
	return payload
}

// generateExpectedXORPayload generates the expected XOR result for source symbols of different lengths.
func generateExpectedXORPayload(sourcePayloads map[protocol.SourceSymbolID][]byte, maxLength int) []byte {
	result := make([]byte, maxLength)
	for _, payload := range sourcePayloads {
		for i := 0; i < len(payload); i++ {
			result[i] ^= payload[i]
		}
	}
	return result
}

func TestXorScheme_recoverSymbolPayloads(t *testing.T) {
	tests := []struct {
		name    string
		block   *block
		want    []byte
		wantErr bool
	}{
		{
			name: "block not recoverable",
			block: &block{
				ssidToSourcePayload: map[protocol.SourceSymbolID][]byte{
					1: {0, 1, 2},
				},
				totNumSourceSymbols: 2,
				totNumRepairSymbols: 1,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "block complete",
			block: &block{
				ssidToSourcePayload: map[protocol.SourceSymbolID][]byte{
					1: {0, 1, 2},
					2: {3, 4, 5},
				},
				totNumSourceSymbols: 2,
				totNumRepairSymbols: 1,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "successful recovery",
			block: &block{
				ssidToSourcePayload: map[protocol.SourceSymbolID][]byte{
					1: {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
				},
				pidToRepairPayload: map[protocol.ParityID][]byte{
					0: {16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 15, 0, 31},
				},
				totNumSourceSymbols:         2,
				totNumRepairSymbols:         1,
				biggestSourceSymbolLenSoFar: 16,
				smallestSSID:                0,
				largestSSID:                 1,
			},
			want:    []byte{16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30},
			wantErr: false,
		},
		{
			name: "successful recovery with big slices",
			block: &block{
				ssidToSourcePayload: map[protocol.SourceSymbolID][]byte{
					1: generateLargePayload(1434, 0x1),
					3: generateLargePayload(1400, 0x4),
				},
				pidToRepairPayload: map[protocol.ParityID][]byte{
					0: func() []byte {
						payload := make([]byte, 1436) // 1434 + 2 bytes for length encoding
						payload = generateExpectedXORPayload(map[protocol.SourceSymbolID][]byte{
							1: generateLargePayload(1434, 0x1),
							2: generateLargePayload(1315, 0x2),
							3: generateLargePayload(1400, 0x4),
						}, len(payload))
						xorLen := uint16(1434 ^ 1315 ^ 1400)

						payload[1434] = byte(xorLen >> 8)
						payload[1435] = byte(xorLen & 0xFF)
						return payload
					}(),
				},
				totNumSourceSymbols:         3,
				totNumRepairSymbols:         1,
				biggestSourceSymbolLenSoFar: 1434,
				smallestSSID:                1,
				largestSSID:                 3,
			},
			want:    generateLargePayload(1315, 0x2),
			wantErr: false,
		},
	}

	scheme := &xorScheme{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scheme.recoverSymbolPayloads(tt.block)
			if (err != nil) != tt.wantErr {
				t.Errorf("recoverSymbolPayloads() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("recoverSymbolPayloads() got = %v, want %v", got, tt.want)
			}
		})
	}
}
