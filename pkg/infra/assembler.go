package infra

import (
	"context"
	"sync"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
)

type Elements struct {
	Proposal   *peer.Proposal
	SignedProp *peer.SignedProposal
	Responses  []*peer.ProposalResponse
	lock       sync.Mutex
	Envelope   *common.Envelope
}

type Assembler struct {
	Signer  Crypto
	Ctx     context.Context
	Raw     chan *Elements
	Signed  []chan *Elements
	ErrorCh chan error
}

func (a *Assembler) sign(e *Elements) (*Elements, error) {
	sprop, err := SignProposal(e.Proposal, a.Signer)
	if err != nil {
		return nil, err
	}
	e.SignedProp = sprop

	return e, nil
}

func (a *Assembler) Start() {
	for {
		select {
		case r := <-a.Raw:
			t, err := a.sign(r)
			if err != nil {
				a.ErrorCh <- err
				return
			}
			for _, v := range a.Signed {
				v <- t
			}
		case <-a.Ctx.Done():
			return
		}
	}
}
