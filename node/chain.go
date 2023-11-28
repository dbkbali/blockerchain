package node

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/dbkbali/blocker/crypto"
	"github.com/dbkbali/blocker/proto"
	"github.com/dbkbali/blocker/types"
)

const initSeed = "b927acba1ee5ebaf030af1a6ac2eb63922942ea39997ad7b2a23754cab1795d3"

type HeaderList struct {
	headers []*proto.Header
}

func NewHeaderList() *HeaderList {
	return &HeaderList{
		headers: []*proto.Header{},
	}
}

func (h *HeaderList) Add(header *proto.Header) {
	h.headers = append(h.headers, header)
}

func (h *HeaderList) Get(index int) *proto.Header {
	if index < h.Height() {
		panic("index out of range")
	}
	return h.headers[index]
}

// Height is one less than the length of the header list
func (h *HeaderList) Height() int {
	return len(h.headers) - 1
}

func (h *HeaderList) Len() int {
	return len(h.headers)
}

type UTXO struct {
	Hash     string
	OutIndex int
	Amount   int64
	Spent    bool
}

type Chain struct {
	txStore    TXStorer
	blockStore BlockStorer
	headers    *HeaderList
	utxoStore  UTXOStorer
}

func NewChain(bs BlockStorer, txStore TXStorer) *Chain {
	chain := &Chain{
		txStore:    txStore,
		blockStore: bs,
		utxoStore:  NewMemoryUTXOStore(),
		headers:    NewHeaderList(),
	}
	chain.addBlock(chain.createGenesisBlock())
	return chain
}

func (c *Chain) Height() int {
	return c.headers.Height()
}

func (c *Chain) AddBlock(b *proto.Block) error {
	if err := c.ValidateBlock(b); err != nil {
		return err
	}

	return c.addBlock(b)
} // Add header to header li}

func (c *Chain) addBlock(b *proto.Block) error {
	c.headers.Add(b.Header)

	for _, tx := range b.Transactions {
		if err := c.txStore.Put(tx); err != nil {
			return err
		}
		hash := hex.EncodeToString(types.HashTransaction(tx))

		for i, output := range tx.Outputs {
			utxo := &UTXO{
				Hash:     hash,
				OutIndex: i,
				Amount:   output.Amount,
				Spent:    false,
			}
			if err := c.utxoStore.Put(utxo); err != nil {
				return err
			}
		}

		for _, input := range tx.Inputs {
			key := fmt.Sprintf("%s_%d", hex.EncodeToString(input.PrevTxHash), input.PrevOutIndex)
			utxo, err := c.utxoStore.Get(key)
			if err != nil {
				return err
			}
			utxo.Spent = true
			if err := c.utxoStore.Put(utxo); err != nil {
				return err
			}
		}

	}

	return c.blockStore.Put(b)
}

func (c *Chain) GetBlockByHash(hash []byte) (*proto.Block, error) {
	hashHex := hex.EncodeToString(hash)
	return c.blockStore.Get(hashHex)
}

func (c *Chain) GetBlockByHeight(height int) (*proto.Block, error) {
	if c.Height() < height {
		return nil, fmt.Errorf("block with height [%d] does not exist", height)
	}
	header := c.headers.Get(height)
	hash := types.HashHeader(header)
	return c.GetBlockByHash(hash)
}

func (c *Chain) ValidateBlock(b *proto.Block) error {
	// validate the signature
	if !types.VerifyBlock(b) {
		return fmt.Errorf("invalid block signature")
	}

	// validate prev hash
	currentBlock, err := c.GetBlockByHeight(c.Height())
	if err != nil {
		return err
	}
	hash := types.HashBlock(currentBlock)
	if !bytes.Equal(hash, b.Header.PrevHash) {
		return fmt.Errorf("invalid previous block hash")
	}
	// TODO
	for _, tx := range b.Transactions {
		if err := c.ValidateTransaction(tx); err != nil {
			return err
		}
	}

	return nil
}

func (c *Chain) ValidateTransaction(tx *proto.Transaction) error {
	// verify signature
	if !types.VerifyTransaction(tx) {
		return fmt.Errorf("invalid transaction signature")
	}
	// validate all inputs unspent
	var (
		nInputs = len(tx.Inputs)
		hash    = hex.EncodeToString(types.HashTransaction(tx))
	)

	sumInputs := int64(0)
	for i := 0; i < nInputs; i++ {
		prevHash := hex.EncodeToString(tx.Inputs[i].PrevTxHash)
		key := fmt.Sprintf("%s_%d", prevHash, i)
		utxo, err := c.utxoStore.Get(key)
		sumInputs += utxo.Amount
		if err != nil {
			return err
		}
		if utxo.Spent {
			return fmt.Errorf("output [%d] of transaction [%s] is already spent", i, hash)
		}
	}

	sumOutputs := int64(0)
	for _, output := range tx.Outputs {
		sumOutputs += output.Amount
	}

	if sumInputs < sumOutputs {
		return fmt.Errorf("insufficient funds unspent (%d) spent (%d)", sumInputs, sumOutputs)
	}
	return nil
}

func (c *Chain) createGenesisBlock() *proto.Block {
	privKey := crypto.NewPrivateKeyFromStringSeed(initSeed)

	block := &proto.Block{
		Header: &proto.Header{
			Version: 1,
		},
	}

	tx := &proto.Transaction{
		Version: 1,
		Inputs:  []*proto.TxInput{},
		Outputs: []*proto.TxOutput{
			{
				Amount:  1000,
				Address: privKey.Public().Address().Bytes(),
			},
		},
	}

	block.Transactions = append(block.Transactions, tx)

	types.SignBlock(privKey, block)

	return block
}
