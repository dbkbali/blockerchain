package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeneratePrivateKey(t *testing.T) {
	privKey := GeneratePrivateKey()
	assert.Equal(t, PrivKeyLen, len(privKey.Bytes()))

	pubKey := privKey.Public()
	assert.Equal(t, PubKeyLen, len(pubKey.Bytes()))
}

func TestPrivateKeyFromString(t *testing.T) {
	var (
		seed       = "30df7d116d04af6b6869fd1407b37866f00c92359bc407290e1dfae6940afa34"
		privKey    = NewPrivateKeyFromString(seed)
		addressStr = "b4358d4516827e70b4066bd441ab6166ea1e015e"
	)

	assert.Equal(t, PrivKeyLen, len(privKey.Bytes()))
	address := privKey.Public().Address()

	assert.Equal(t, addressStr, address.String())
}

func TestPrivateKeySign(t *testing.T) {
	privKey := GeneratePrivateKey()
	pubKey := privKey.Public()
	msg := []byte("hello world")

	sig := privKey.Sign(msg)
	assert.True(t, sig.Verify(msg, pubKey))

	// Invalid message
	assert.False(t, sig.Verify([]byte("hello world!"), pubKey))

	// Invalid public key
	altKey := GeneratePrivateKey().Public()
	assert.False(t, sig.Verify(msg, altKey))
}

func TestPublicKeyAddress(t *testing.T) {
	privKey := GeneratePrivateKey()
	pubKey := privKey.Public()
	address := pubKey.Address()

	assert.Equal(t, AddressLen, len(address.Bytes()))
}
