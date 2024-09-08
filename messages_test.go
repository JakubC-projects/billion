package diamond

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeBoxCheckRequest(t *testing.T) {
	data := []byte{0x01, 0x01, 0x00, 0x04}
	buf := bytes.NewBuffer(data)

	var m MessageBoxCheckRequest
	err := m.Decode(buf)
	assert.NoError(t, err)
	assert.Equal(t, m, MessageBoxCheckRequest{Coordinates{257, 4}})
}

// func TestDecodeMessageChunkRequest(t *testing.T) {
// 	data := []byte{
// 		0x01, 0x01, 0x00, 0x04,
// 		0x01, 0x05, 0x00, 0x03,
// 	}
// 	buf := bytes.NewBuffer(data)

// 	var m MessageChunkRequest
// 	err := m.Decode(buf)
// 	assert.NoError(t, err)
// 	assert.Equal(t, m, MessageChunkRequest{Chunks: []Coordinates{
// 		{257, 4},
// 		{261, 3},
// 	}})
// }

func TestEncodeGameStats(t *testing.T) {
	var gameStats = MessageGameStats{
		UncoveredCount: 10,
		OnlineCount:    5,
		GoldPositions: []Coordinates{
			{1, 2},
		},
	}

	expectedResult := []byte{
		0x00, 0x00, 0x00, 0x0a, // UncoveredCount
		0x00, 0x00, 0x00, 0x05, // OnlineCount

		//Gold positions
		0x00, 0x01, 0x00, 0x02,
	}

	buf := &bytes.Buffer{}

	err := gameStats.Encode(buf)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, buf.Bytes())
}
