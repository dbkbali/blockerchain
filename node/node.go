package node

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/dbkbali/blocker/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
)

type Node struct {
	version    string
	listenAddr string
	logger     *zap.SugaredLogger

	peerLock sync.RWMutex
	peers    map[proto.NodeClient]*proto.HandshakeRequest
	proto.UnimplementedNodeServer
}

func NewNode() *Node {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.TimeKey = ""
	logger, _ := loggerConfig.Build()
	return &Node{
		version: "blocker-1",
		peers:   make(map[proto.NodeClient]*proto.HandshakeRequest),
		logger:  logger.Sugar(),
	}
}

func (n *Node) Start(listenAddr string, bootstrapNodes []string) error {
	n.listenAddr = listenAddr
	var (
		opts       = []grpc.ServerOption{}
		grpcServer = grpc.NewServer(opts...)
	)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	proto.RegisterNodeServer(grpcServer, n)

	n.logger.Infow("node started...", "port:", n.listenAddr)

	// bootstrap network with known bootstrapNodes
	if len(bootstrapNodes) > 0 {
		go n.bootstrapNetwork(bootstrapNodes)
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
	fmt.Println("Received transaction from", peer.Addr)
	return &proto.Ack{}, nil
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
		"we", n.listenAddr,
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
		n.logger.Debugw("dialing remote node", "we", n.listenAddr, "remote", addr)

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
		ListenAddr: n.listenAddr,
		PeerList:   n.getPeerList(),
	}
}

func (n *Node) canConnectWith(addr string) bool {
	if n.listenAddr == addr {
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
