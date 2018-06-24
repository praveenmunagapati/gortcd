package allocator

import (
	"errors"
	"net"
	"sync"

	"go.uber.org/zap"

	"github.com/gortc/turn"
)

type NetAllocation struct {
	Addr  Addr
	Proto turn.Protocol
	Conn  net.PacketConn
}

func (n *NetAllocation) Close() error {
	err := n.Conn.Close()
	n.Conn = nil
	n.Addr = Addr{}
	n.Proto = 0
	return err
}

// NetAllocator allocates ports.
type NetAllocator struct {
	allocsMux sync.RWMutex
	allocs    []NetAllocation
	newAllocs []NetAllocation
	ports     NetPortAllocator

	log         *zap.Logger
	defaultAddr string
}

type NetPortAllocator interface {
	AllocatePort(proto turn.Protocol, network, defaultAddr string) (NetAllocation, error)
}

func (a *NetAllocator) New(proto turn.Protocol) (Addr, net.PacketConn, error) {
	n, err := a.ports.AllocatePort(proto, "udp4", a.defaultAddr)
	if err != nil {
		return Addr{}, nil, err
	}
	a.allocsMux.Lock()
	a.allocs = append(a.allocs, n)
	a.allocsMux.Unlock()
	return n.Addr, n.Conn, nil
}

func (a *NetAllocator) Remove(addr Addr, proto turn.Protocol) error {
	var (
		toRemove []NetAllocation // TODO: optimize heap alloc
	)

	a.allocsMux.Lock()
	for _, alloc := range a.allocs {
		if alloc.Proto != proto {
			a.newAllocs = append(a.newAllocs, alloc)
			continue
		}
		if !addr.Equal(alloc.Addr) {
			a.newAllocs = append(a.newAllocs, alloc)
			continue
		}
		toRemove = append(toRemove, alloc)
	}
	if len(toRemove) == 0 {
		a.newAllocs = a.newAllocs[:0]
		a.allocsMux.Unlock()
		return nil
	}
	n := copy(a.allocs, a.newAllocs)
	a.allocs = a.allocs[:n]
	a.newAllocs = a.newAllocs[:0]
	a.allocsMux.Unlock()

	for _, r := range toRemove {
		if err := r.Close(); err != nil {
			a.log.Error("failed to remove", zap.Error(err))
		}
	}
	return nil
}

func NewNetAllocator(l *zap.Logger, addr net.Addr, ports NetPortAllocator) (*NetAllocator, error) {
	var defaultAddr string
	switch tAddr := addr.(type) {
	case *net.UDPAddr:
		defaultAddr = tAddr.IP.String() + ":0"
	default:
		return nil, errors.New("unsupported addr")
	}
	a := NetAllocator{
		log:         l,
		defaultAddr: defaultAddr,
		ports:       ports,
	}
	return &a, nil
}
