package tcp

import (
	"net"
	"time"
)

func SendRequest(address string, payload []byte) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	if err := WriteMessage(conn, payload); err != nil {
		return nil, err
	}

	return ReadMessage(conn)
}
