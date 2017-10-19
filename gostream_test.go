package gostream

import (
	"bufio"
	"context"
	"io/ioutil"
	"testing"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	swarm "github.com/libp2p/go-libp2p-swarm"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	multiaddr "github.com/multiformats/go-multiaddr"
)

// newHost illustrates how to build a libp2p host with secio using
// a randomly generated key-pair
func newHost(t *testing.T, listen multiaddr.Multiaddr) host.Host {
	priv, pub, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	ps := peerstore.NewPeerstore()
	err = ps.AddPubKey(pid, pub)
	if err != nil {
		t.Fatal(err)
	}
	err = ps.AddPrivKey(pid, priv)
	if err != nil {
		t.Fatal(err)
	}

	network, err := swarm.NewNetwork(
		context.Background(),
		[]multiaddr.Multiaddr{listen},
		pid,
		ps,
		nil)

	if err != nil {
		t.Fatal(err)
	}

	host := bhost.New(network)
	return host
}

func TestServerClient(t *testing.T) {
	m1, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10000")
	m2, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/10001")
	srvHost := newHost(t, m1)
	clientHost := newHost(t, m2)
	defer srvHost.Close()
	defer clientHost.Close()

	srvHost.Peerstore().AddAddrs(clientHost.ID(), clientHost.Addrs(), peerstore.PermanentAddrTTL)
	clientHost.Peerstore().AddAddrs(srvHost.ID(), srvHost.Addrs(), peerstore.PermanentAddrTTL)

	var tag protocol.ID = "/testitytest"

	go func() {
		listener, err := Listen(srvHost, tag)
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()

		if listener.Addr().String() != srvHost.ID().Pretty() {
			t.Fatal("bad listener address")
		}

		servConn, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer servConn.Close()

		reader := bufio.NewReader(servConn)
		msg, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if string(msg) != "is libp2p awesome?\n" {
			t.Fatalf("Bad incoming message: %s", msg)
		}

		_, err = servConn.Write([]byte("yes it is"))
		if err != nil {
			t.Fatal(err)
		}
	}()

	clientConn, err := Dial(clientHost, srvHost.ID(), tag)
	if err != nil {
		t.Fatal(err)
	}

	if clientConn.LocalAddr().String() != clientHost.ID().Pretty() {
		t.Fatal("Bad LocalAddr")
	}

	if clientConn.RemoteAddr().String() != srvHost.ID().Pretty() {
		t.Fatal("Bad RemoteAddr")
	}

	if clientConn.LocalAddr().Network() != Network {
		t.Fatal("Bad Network()")
	}

	err = clientConn.SetDeadline(time.Now().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	err = clientConn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	err = clientConn.SetWriteDeadline(time.Now().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}

	_, err = clientConn.Write([]byte("is libp2p awesome?\n"))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := ioutil.ReadAll(clientConn)
	if err != nil {
		t.Fatal(err)
	}

	if string(resp) != "yes it is" {
		t.Errorf("Bad response: %s", resp)
	}

	err = clientConn.Close()
	if err != nil {
		t.Fatal(err)
	}
}
