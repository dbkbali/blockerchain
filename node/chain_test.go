package node

import (
	"testing"

	"github.com/dbkbali/blocker/crypto"
	"github.com/dbkbali/blocker/proto"
	"github.com/dbkbali/blocker/types"
	"github.com/dbkbali/blocker/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func randomBlock(t *testing.T, chain *Chain) *proto.Block {
	privKey := crypto.GeneratePrivateKey()
	block := util.RandomBlock()
	prevBlock, err := chain.GetBlockByHeight(chain.Height())
	require.Nil(t, err)
	block.Header.PrevHash = types.HashBlock(prevBlock)
	types.SignBlock(privKey, block)
	return block
}

func TestNewChain(t *testing.T) {
	chain := NewChain(NewMemoryBlockStore(), NewMemoryTXStore())
	assert.Equal(t, 0, chain.Height())
	_, err := chain.GetBlockByHeight(0)
	assert.Nil(t, err)
}

func TestChainHeight(t *testing.T) {
	chain := NewChain(NewMemoryBlockStore(), NewMemoryTXStore())
	for i := 0; i < 10; i++ {
		block := randomBlock(t, chain)

		require.Nil(t, chain.AddBlock(block))
		require.Equal(t, i+1, chain.Height())
	}
}

func TestAddBlock(t *testing.T) {
	chain := NewChain(NewMemoryBlockStore(), NewMemoryTXStore())

	for i := 0; i < 100; i++ {
		block := randomBlock(t, chain)

		blockHash := types.HashBlock(block)

		require.Nil(t, chain.AddBlock(block))

		fetchedBlock, err := chain.GetBlockByHash(blockHash)
		require.Nil(t, err)
		require.Equal(t, block, fetchedBlock)

		fetchedBlockByHeight, err := chain.GetBlockByHeight(i + 1)
		require.Nil(t, err)
		require.Equal(t, block, fetchedBlockByHeight)
	}
}

func TestAddBlockWithTxInsufficientFunds(t *testing.T) {
	var (
		chain     = NewChain(NewMemoryBlockStore(), NewMemoryTXStore())
		block     = randomBlock(t, chain)
		privKey   = crypto.NewPrivateKeyFromStringSeed(initSeed)
		recipient = crypto.GeneratePrivateKey().Public().Address().Bytes()
	)

	prevTx, err := chain.txStore.Get("7c66fa0ecedf3f4748bba3694df77f8b86a197559d9810dde357b78a8badcc8a")
	assert.Nil(t, err)

	inputs := []*proto.TxInput{
		{
			PrevTxHash:   types.HashTransaction(prevTx),
			PrevOutIndex: 0,
			PublicKey:    privKey.Public().Bytes(),
		},
	}
	outputs := []*proto.TxOutput{
		{
			Amount:  1001,
			Address: recipient,
		},
	}
	tx := &proto.Transaction{
		Version: 1,
		Inputs:  inputs,
		Outputs: outputs,
	}

	sig := types.SignTransaction(privKey, tx)
	tx.Inputs[0].Signature = sig.Bytes()
	block.Transactions = append(block.Transactions, tx)
	require.NotNil(t, chain.AddBlock(block))

}
func TestAddBlockWithTxs(t *testing.T) {
	var (
		chain     = NewChain(NewMemoryBlockStore(), NewMemoryTXStore())
		block     = randomBlock(t, chain)
		privKey   = crypto.NewPrivateKeyFromStringSeed(initSeed)
		recipient = crypto.GeneratePrivateKey().Public().Address().Bytes()
	)

	prevTx, err := chain.txStore.Get("7c66fa0ecedf3f4748bba3694df77f8b86a197559d9810dde357b78a8badcc8a")
	assert.Nil(t, err)

	inputs := []*proto.TxInput{
		{
			PrevTxHash:   types.HashTransaction(prevTx),
			PrevOutIndex: 0,
			PublicKey:    privKey.Public().Bytes(),
		},
	}
	outputs := []*proto.TxOutput{
		{
			Amount:  100,
			Address: recipient,
		},
		{
			Amount:  900,
			Address: privKey.Public().Address().Bytes(),
		},
	}
	tx := &proto.Transaction{
		Version: 1,
		Inputs:  inputs,
		Outputs: outputs,
	}

	sig := types.SignTransaction(privKey, tx)
	tx.Inputs[0].Signature = sig.Bytes()
	block.Transactions = append(block.Transactions, tx)
	require.Nil(t, chain.AddBlock(block))

}
