package billion

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
)

type MessageType uint8

const (
	MessageTypeBoxCheckRequest = 0
	MessageTypeBoxesUncovered  = 1

	MessageTypeChunkRequest   = 2
	MessageTypeChunksResponse = 3

	MessageTypeGameStats = 4
	MessageTypeError     = 255
)

type Decoder interface {
	Decode(io.Reader) error
}

type Encoder interface {
	Encode(io.Writer) error
}

type MessageBoxCheckRequest struct {
	Coordinates Coordinates
}

func (m *MessageBoxCheckRequest) Decode(r io.Reader) error {
	return m.Coordinates.Decode(r)
}

type MessageBoxesUncovered struct {
	Boxes []Coordinates
}

func (m *MessageBoxesUncovered) Encode(w io.Writer) error {
	_, err := w.Write([]byte{MessageTypeBoxesUncovered})
	if err != nil {
		return err
	}
	for _, c := range m.Boxes {
		err := c.Encode(w)
		if err != nil {
			return err
		}
	}
	return nil
}

type MessageChunkRequest struct {
	Chunks []Coordinates
}

func (m *MessageChunkRequest) Decode(r io.Reader) error {
	var len uint16
	err := binary.Read(r, binary.BigEndian, &len)
	if err != nil {
		return err
	}
	m.Chunks = slices.Grow(m.Chunks, int(len))
	m.Chunks = m.Chunks[:len]

	for i := range len {
		err := m.Chunks[i].Decode(r)
		if err != nil {
			return err
		}
	}

	return nil
}

type MessageChunksResponse struct {
	Chunks []ChunkResponse
}

func (c MessageChunksResponse) Encode(w io.Writer) error {
	_, err1 := w.Write([]byte{MessageTypeChunksResponse})
	err2 := encodeSlice(w, c.Chunks)
	return errors.Join(err1, err2)
}

type ChunkResponse struct {
	Coordinates Coordinates
	Chunk       Chunk
}

func (c ChunkResponse) Encode(w io.Writer) error {
	err := c.Coordinates.Encode(w)
	if err != nil {
		return err
	}
	for y := 0; y < len(c.Chunk); y++ {
		for x := 0; x < len(c.Chunk[y]); x++ {
			err := binary.Write(w, binary.BigEndian, c.Chunk[y][x])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type MessageGameStats struct {
	UncoveredCount uint32
	OnlineCount    uint32
	GoldPositions  []Coordinates

	RecentlyUncovered []Uncover
}

func (m MessageGameStats) Encode(w io.Writer) error {
	_, err1 := w.Write([]byte{MessageTypeGameStats})
	err2 := binary.Write(w, binary.BigEndian, m.UncoveredCount)
	err3 := binary.Write(w, binary.BigEndian, m.OnlineCount)

	err4 := encodeSlice(w, m.GoldPositions)
	err5 := encodeSlice(w, m.RecentlyUncovered)

	return errors.Join(err1, err2, err3, err4, err5)
}

type MessageError struct {
	Message string
}

type Uncover struct {
	Coordinates
	TickTiming uint8
}

func (m Uncover) Encode(w io.Writer) error {
	err1 := m.Coordinates.Encode(w)
	err2 := binary.Write(w, binary.BigEndian, m.TickTiming)
	return errors.Join(err1, err2)
}

type Coordinates struct {
	X uint16
	Y uint16
}

func (m *Coordinates) Decode(r io.Reader) error {
	err1 := binary.Read(r, binary.BigEndian, &m.X)
	if errors.Is(err1, io.EOF) {
		return err1
	}
	err2 := binary.Read(r, binary.BigEndian, &m.Y)
	return errors.Join(err1, err2)
}

func (m Coordinates) Encode(w io.Writer) error {
	err1 := binary.Write(w, binary.BigEndian, m.X)
	err2 := binary.Write(w, binary.BigEndian, m.Y)
	return errors.Join(err1, err2)
}

func encodeSlice[T Encoder](w io.Writer, slice []T) error {
	err := binary.Write(w, binary.BigEndian, uint16(len(slice)))
	if err != nil {
		return err
	}
	for _, e := range slice {
		err := e.Encode(w)
		if err != nil {
			return err
		}
	}

	return nil
}

func parseMessageType(r io.Reader) (MessageType, error) {
	var messageType MessageType
	err := binary.Read(r, binary.BigEndian, &messageType)
	if err != nil {
		return messageType, fmt.Errorf("cannot read message type: %w", err)
	}
	return messageType, nil
}
