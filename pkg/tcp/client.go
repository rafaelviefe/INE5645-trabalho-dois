package tcp

import (
	"net"
)

func SendRequest(address string, payload []byte) ([]byte, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := WriteMessage(conn, payload); err != nil {
		return nil, err
	}

	return ReadMessage(conn)
}