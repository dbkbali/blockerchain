package node

import (
	"context"
	"encoding/hex"
	"net"
	"sync"
	"time"

	"github.com/dbkbali/blocker/crypto"
	"github.com/dbkbali/blocker/proto"
	"github.com/dbkbali/blocker/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
)

const blockTime = time.Second * 5

type Mempool struct {
	lock sync.RWMutex
	txx  map[string]*proto.Transaction
}

func NewMempool() *Mempool {
	return &Mempool{
		txx: make(map[string]*proto.Transaction),
	}
}

func (pool *Mempool) Clear() []*proto.Transaction {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	txx := make([]*proto.Transaction, len(pool.txx))
	iter := 0
	for k, tx := range pool.txx {
		delete(pool.txx, k)
		txx[iter] = tx
		iter++
	}
	return txx
}

func (pool *Mempool) Len() int {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	return len(pool.txx)
}

func (m *Mempool) Has(tx *proto.Transaction) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	hash := hex.EncodeToString(types.HashTransaction(tx))
	_, ok := m.txx[hash]
	return ok
}

func (m *Mempool) Add(tx *proto.Transaction) bool {
	if m.Has(tx) {
		return false
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	hash := hex.EncodeToString(types.HashTransaction(tx))
	m.txx[hash] = tx
	return true
}

type ServerConfig struct {
	Version    string
	ListenAddr string
	PrivateKey *crypto.PrivateKey
}

type Node struct {
	ServerConfig
	logger *zap.SugaredLogger

	peerLock sync.RWMutex
	peers    map[proto.NodeClient]*proto.HandshakeRequest
	mempool  *Mempool
	proto.UnimplementedNodeServer
}

func NewNode(cfg ServerConfig) *Node {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.TimeKey = ""
	logger, _ := loggerConfig.Build()
	return &Node{
		ServerConfig: cfg,
		peers:        make(map[proto.NodeClient]*proto.HandshakeRequest),
		logger:       logger.Sugar(),
		mempool:      NewMempool(),
	}
}

func (n *Node) Start(listenAddr string, bootstrapNodes []string) error {
	n.ListenAddr = listenAddr
	var (
		opts       = []grpc.ServerOption{}
		grpcServer = grpc.NewServer(opts...)
	)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	proto.RegisterNodeServer(grpcServer, n)

	n.logger.Infow("node started...", "port:", n.ListenAddr)

	// bootstrap network with known bootstrapNodes
	if len(bootstrapNodes) > 0 {
		go n.bootstrapNetwork(bootstrapNodes)
	}

	if n.PrivateKey != nil {
		go n.validatorLoop()
	}

	return grpcServer.Serve(ln)
}

func (n *Node) Handshake(ctx context.Context, req *proto.HandshakeRequest) (*proto.HandshakeRequest, error) {
	c, err := makeNodeClient(req.ListenAddr)
	if err != nil {
		return nil, err
	}

	n.addPeer(c, req)

	return n.getHandshakeRequest(), nil
}

func (n *Node) HandleTransaction(ctx context.Context, tx *proto.Transaction) (*proto.Ack, error) {
	peer, _ := peer.FromContext(ctx)
	hash := hex.EncodeToString(types.HashTransaction(tx))

	if n.mempool.Add(tx) {
		n.logger.Debugw("Received tx", "from", peer.Addr, "hash", hash, "we", n.ListenAddr)
		go func() {
			if err := n.broadcast(tx); err != nil {
				n.logger.Errorw("broadcast failure", "err", err)
			}
		}()
	}

	return &proto.Ack{}, nil
}

func (n *Node) validatorLoop() {
	n.logger.Infow("starting validator loop", "pubkey", n.PrivateKey.Public(), "blockTime", blockTime)
	ticker := time.NewTicker(blockTime)
	for {
		<-ticker.C

		txx := n.mempool.Clear()

		n.logger.Debugw("time to create a new block", "lenTx", len(txx))
	}

}
func (n *Node) broadcast(msg any) error {
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()

	for peer := range n.peers {
		switch v := msg.(type) {
		case *proto.Transaction:
			_, err := peer.HandleTransaction(context.Background(), v)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *Node) addPeer(peer proto.NodeClient, handshake *proto.HandshakeRequest) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()

	//Logic to check if peer accepts or drop connection

	n.peers[peer] = handshake

	// connect to all Peers in the received peerList
	if len(handshake.PeerList) > 0 {
		go n.bootstrapNetwork(handshake.PeerList)
	}

	n.logger.Debugw("new peer successfully connected",
		"we", n.ListenAddr,
		"remoteNode", handshake.ListenAddr,
		"height", handshake.Height)
}

func (n *Node) deletePeer(peer proto.NodeClient) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()

	delete(n.peers, peer)
}

func (n *Node) bootstrapNetwork(bootNodes []string) error {
	for _, addr := range bootNodes {
		if !n.canConnectWith(addr) {
			continue
		}
		n.logger.Debugw("dialing remote node", "we", n.ListenAddr, "remote", addr)

		c, h, err := n.dialRemoteNode(addr)
		if err != nil {
			return err
		}
		n.addPeer(c, h)
	}
	return nil
}

func (n *Node) dialRemoteNode(addr string) (proto.NodeClient, *proto.HandshakeRequest, error) {
	c, err := makeNodeClient(addr)
	if err != nil {
		return nil, nil, err
	}

	h, err := c.Handshake(context.Background(), n.getHandshakeRequest())
	if err != nil {
		return nil, nil, err
	}

	return c, h, nil
}

func (n *Node) getHandshakeRequest() *proto.HandshakeRequest {
	return &proto.HandshakeRequest{
		Version:    "blocker-01",
		Height:     0,
		ListenAddr: n.ListenAddr,
		PeerList:   n.getPeerList(),
	}
}

func (n *Node) canConnectWith(addr string) bool {
	if n.ListenAddr == addr {
		return false
	}

	connectedPeers := n.getPeerList()
	for _, connectedAddr := range connectedPeers {
		if addr == connectedAddr {
			return false
		}
	}

	return true
}

func (n *Node) getPeerList() []string {
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()

	peers := []string{}
	for _, handshakeRequest := range n.peers {
		peers = append(peers, handshakeRequest.ListenAddr)
	}
	return peers
}

func makeNodeClient(listenAddr string) (proto.NodeClient, error) {
	conn, err := grpc.Dial(listenAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return proto.NewNodeClient(conn), nil
}
