package gostream

import (
	"context"
	"net"

	host "github.com/libp2p/go-libp2p-host"
	pnet "github.com/libp2p/go-libp2p-net"
	protocol "github.com/libp2p/go-libp2p-protocol"
)

// Listener is an implementation of net.Listener which handles
// http-tagged streams from a libp2p connection.
// A listener can be built with Listen()
type Listener struct {
	host     host.Host
	ctx      context.Context
	tag      protocol.ID
	cancel   func()
	streamCh chan pnet.Stream
}

// Accept returns a connection from this listener. It blocks if there
// are no connections.
func (l *Listener) Accept() (net.Conn, error) {
	select {
	case s := <-l.streamCh:
		return NewConn(s), nil
	case <-l.ctx.Done():
		return nil, l.ctx.Err()
	}
}

// Close terminates this listener. It will no longer handle any
// incoming streams
func (l *Listener) Close() error {
	l.cancel()
	l.host.RemoveStreamHandler(l.tag)
	return nil
}

// Addr returns the address for this listener, which is its libp2p Peer ID.
func (l *Listener) Addr() net.Addr {
	return &Addr{l.host.ID()}
}

// Listen creates a new listener ready to accept streams received by a host.
func Listen(h host.Host, tag protocol.ID) (net.Listener, error) {
	ctx, cancel := context.WithCancel(context.Background())

	l := &Listener{
		host:     h,
		ctx:      ctx,
		cancel:   cancel,
		tag:      tag,
		streamCh: make(chan pnet.Stream),
	}

	h.SetStreamHandler(tag, func(s pnet.Stream) {
		select {
		case l.streamCh <- s:
		case <-ctx.Done():
			s.Close()
		}
	})

	return l, nil
}
