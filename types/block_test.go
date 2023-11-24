package types

import (
	"testing"

	"github.com/dbkbali/blocker/crypto"
	"github.com/dbkbali/blocker/util"
	"github.com/stretchr/testify/assert"
)

func TestSignBlock(t *testing.T) {
	var (
		privKey = crypto.GeneratePrivateKey()
		block   = util.RandomBlock()
		pubKey  = privKey.Public()
	)
	sig := SignBlock(privKey, block)
	assert.Equal(t, 64, len(sig.Bytes()))
	assert.True(t, sig.Verify(HashBlock(block), pubKey))
}

func TestHashBlock(t *testing.T) {
	block := util.RandomBlock()
	hash := HashBlock(block)
	assert.Equal(t, 32, len(hash))
}
