package plugin

import (
	"encoding/binary"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
)

// MaxMessageSize is the maximum allowed size for a single framed protobuf
// message (16 MB).
const MaxMessageSize = 16 * 1024 * 1024

// WriteMessage marshals a protobuf message and writes it to w using
// length-delimited framing: [4 bytes big-endian uint32 length][N bytes payload].
func WriteMessage(w io.Writer, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if len(data) > MaxMessageSize {
		return fmt.Errorf("message size %d exceeds maximum %d", len(data), MaxMessageSize)
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(data)))
	if _, err := w.Write(header[:]); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}
	return nil
}

// ReadMessage reads a length-delimited protobuf message from r and unmarshals
// it into msg. The framing format must match WriteMessage.
func ReadMessage(r io.Reader, msg proto.Message) error {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	size := binary.BigEndian.Uint32(header[:])
	if size > MaxMessageSize {
		return fmt.Errorf("message size %d exceeds maximum %d", size, MaxMessageSize)
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return fmt.Errorf("read payload: %w", err)
	}
	if err := proto.Unmarshal(data, msg); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}
