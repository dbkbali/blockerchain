package node

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/dbkbali/blocker/proto"
	"github.com/dbkbali/blocker/types"
)

type TXStorer interface {
	Put(*proto.Transaction) error
	Get(string) (*proto.Transaction, error)
}

type MemoryTXStore struct {
	lock sync.RWMutex
	txx  map[string]*proto.Transaction
}

func NewMemoryTXStore() *MemoryTXStore {
	return &MemoryTXStore{
		txx: make(map[string]*proto.Transaction),
	}
}

func (s *MemoryTXStore) Put(tx *proto.Transaction) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	hash := hex.EncodeToString(types.HashTransaction(tx))
	s.txx[hash] = tx
	return nil
}

func (s *MemoryTXStore) Get(hash string) (*proto.Transaction, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	tx, ok := s.txx[hash]
	if !ok {
		return nil, fmt.Errorf("transaction with hash [%s] does not exist", hash)
	}
	return tx, nil
}

type BlockStorer interface {
	Put(*proto.Block) error
	Get(string) (*proto.Block, error)
}

type MemoryBlockStore struct {
	lock   sync.RWMutex
	blocks map[string]*proto.Block
}

func NewMemoryBlockStore() *MemoryBlockStore {
	return &MemoryBlockStore{
		blocks: make(map[string]*proto.Block),
	}
}

func (m *MemoryBlockStore) Put(block *proto.Block) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	hash := hex.EncodeToString(types.HashBlock(block))
	m.blocks[hash] = block
	return nil
}

func (m *MemoryBlockStore) Get(hash string) (*proto.Block, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	block, ok := m.blocks[hash]
	if !ok {
		return nil, fmt.Errorf("block with hash [%s] does not exist", hash)
	}
	return block, nil
}
