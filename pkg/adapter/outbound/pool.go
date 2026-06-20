package outbound

import (
	"fmt"
	"sync"
	"time"
)

type Pool struct {
	addresses []string
	cursor    int
	cooldown  time.Duration
	failedAt  map[string]time.Time
	mu        sync.Mutex
}

func NewPool(addrs []string, cooldown time.Duration) *Pool {
	return &Pool{
		addresses: addrs,
		cursor:    0,
		cooldown:  cooldown,
		failedAt:  make(map[string]time.Time),
	}
}

func (p *Pool) Next() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.addresses) == 0 {
		return "", fmt.Errorf("pool: nenhum endereco configurado")
	}

	start := p.cursor
	now := time.Now()

	for i := 0; i < len(p.addresses); i++ {
		idx := (start + i) % len(p.addresses)
		addr := p.addresses[idx]

		if failTime, ok := p.failedAt[addr]; ok {
			if now.Before(failTime.Add(p.cooldown)) {
				continue
			}
			delete(p.failedAt, addr)
		}

		p.cursor = (idx + 1) % len(p.addresses)
		return addr, nil
	}

	return "", fmt.Errorf("pool: todos os %d enderecos estao em cooldown", len(p.addresses))
}

func (p *Pool) MarkUnhealthy(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failedAt[addr] = time.Now()
}
