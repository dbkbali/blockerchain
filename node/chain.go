package node

import (
	"encoding/hex"
	"fmt"

	"github.com/dbkbali/blocker/proto"
	"github.com/dbkbali/blocker/types"
)

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

type Chain struct {
	blockStore BlockStorer
	headers    *HeaderList
}

func NewChain(bs BlockStorer) *Chain {
	return &Chain{
		blockStore: bs,
		headers:    NewHeaderList(),
	}
}

func (c *Chain) Height() int {
	return c.headers.Height()
}

func (c *Chain) AddBlock(b *proto.Block) error {
	// Add header to header list
	c.headers.Add(b.Header)
	// validation TODO
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
