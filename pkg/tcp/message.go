package tcp

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

var ErrPayloadTooLarge = errors.New("payload exceeds maximum allowed size")

const maxPayloadSize = 10 * 1024 * 1024

func ReadMessage(conn net.Conn) ([]byte, error) {
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lengthBuf); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length > maxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	msgBuf := make([]byte, length)
	if _, err := io.ReadFull(conn, msgBuf); err != nil {
		return nil, err
	}

	return msgBuf, nil
}

func WriteMessage(conn net.Conn, payload []byte) error {
	length := uint32(len(payload))
	lengthBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBuf, length)

	if _, err := conn.Write(lengthBuf); err != nil {
		return err
	}

	_, err := conn.Write(payload)
	return err
}
