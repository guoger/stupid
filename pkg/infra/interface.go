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

/*
type consmuer and producer interface
as Tape major as Producer and Consumer pattern
define an interface here as Worker with start here
as for #56 and #174,in cli imp adjust sequence of P&C impl to control workflow.
*/
type Worker interface {
	Start()
}
