package remote_executor

import (
	"fmt"
	"sync"

	"github.com/caiflower/common-tools/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type connEntry struct {
	conn *grpc.ClientConn
	refs int
}

type ConnPool struct {
	mu    sync.Mutex
	conns map[string]*connEntry
}

func NewConnPool() *ConnPool {
	return &ConnPool{conns: make(map[string]*connEntry)}
}

func (p *ConnPool) GetConn(address string) (*grpc.ClientConn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.conns[address]
	if ok {
		entry.refs++
		return entry.conn, nil
	}

	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}

	p.conns[address] = &connEntry{conn: conn, refs: 1}
	return conn, nil
}

func (p *ConnPool) Release(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entry, ok := p.conns[address]
	if !ok {
		return
	}
	entry.refs--
	if entry.refs <= 0 {
		entry.conn.Close()
		delete(p.conns, address)
	}
}

func (p *ConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	logger.Info("remote_executor closing all connections")
	for addr, entry := range p.conns {
		entry.conn.Close()
		delete(p.conns, addr)
	}
}
