package node

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/dbkbali/blocker/proto"
	"github.com/dbkbali/blocker/types"
)

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
