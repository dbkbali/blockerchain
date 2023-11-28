package main

import (
	"context"
	"log"
	"time"

	"github.com/dbkbali/blocker/crypto"
	"github.com/dbkbali/blocker/node"
	"github.com/dbkbali/blocker/proto"
	"github.com/dbkbali/blocker/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	makeNode(":3000", []string{}, true)
	time.Sleep(1 * time.Second)
	makeNode(":4000", []string{":3000"}, false)
	time.Sleep(1 * time.Second)
	makeNode(":3002", []string{":4000"}, false)

	for {
		time.Sleep(time.Second)
		makeTransaction()
	}

}

func makeNode(listenAddr string, bootstrapNodes []string, isValidator bool) *node.Node {
	cfg := &node.ServerConfig{
		Version:    "0.0.1",
		ListenAddr: listenAddr,
	}
	if isValidator {
		cfg.PrivateKey = crypto.GeneratePrivateKey()
	}

	n := node.NewNode(*cfg)
	go n.Start(listenAddr, bootstrapNodes)
	return n
}

func makeTransaction() {
	client, err := grpc.Dial(":3000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	c := proto.NewNodeClient(client)
	privKey := crypto.GeneratePrivateKey()
	tx := &proto.Transaction{
		Version: 1,
		Inputs: []*proto.TxInput{
			{
				PrevTxHash:   util.RandomHash(),
				PrevOutIndex: 0,
				PublicKey:    privKey.Bytes(),
			},
		},
		Outputs: []*proto.TxOutput{
			{
				Amount:  99,
				Address: privKey.Public().Address().Bytes(),
			},
		}}

	_, err = c.HandleTransaction(context.TODO(), tx)
	if err != nil {
		log.Fatalf("Error when calling Handshake: %v", err)
	}
}
