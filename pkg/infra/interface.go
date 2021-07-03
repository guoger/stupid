package infra

import (
	"github.com/hyperledger/fabric-protos-go/common"
)

/*
to do for #127 SM crypto
just need to do an impl for this interface and replace
and impl a function for func (c Config) LoadCrypto() (*CryptoImpl, error) {
as generator
*/
type Crypto interface {
	NewSignatureHeader() (*common.SignatureHeader, error)
	Serialize() ([]byte, error)
	Sign(message []byte) ([]byte, error)
}
