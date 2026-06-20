package outbound

import (
	"encoding/json"
	"trading-saga/pkg/tcp"
)

type TcpPublisher struct {
	brokerAddr string
}

func NewPublisher(brokerAddr string) *TcpPublisher {
	return &TcpPublisher{brokerAddr: brokerAddr}
}

func (p *TcpPublisher) Publish(channel string, data any) error {
	envelope := map[string]any{
		"action":  "pub",
		"channel": channel,
		"data":    data,
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	_, err = tcp.SendRequest(p.brokerAddr, payload)
	return err
}
