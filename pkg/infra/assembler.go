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
	Signer Crypto
}

func (a *Assembler) assemble(e *Elements) (*Elements, error) {
	env, err := CreateSignedTx(e.Proposal, a.Signer, e.Responses)
	if err != nil {
		return nil, err
	}
	e.Envelope = env
	return e, nil
}

func (a *Assembler) sign(e *Elements) (*Elements, error) {
	sprop, err := SignProposal(e.Proposal, a.Signer)
	if err != nil {
		return nil, err
	}
	e.SignedProp = sprop

	return e, nil
}

func (a *Assembler) StartSigner(ctx context.Context, raw chan *Elements, signed []chan *Elements, errorCh chan error) {
	for {
		select {
		case r := <-raw:
			t, err := a.sign(r)
			if err != nil {
				errorCh <- err
				return
			}
			for _, v := range signed {
				v <- t
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *Assembler) StartIntegrator(ctx context.Context, processed, envs chan *Elements, errorCh chan error) {
	for {
		select {
		case p := <-processed:
			e, err := a.assemble(p)
			if err != nil {
				errorCh <- err
				return
			}
			envs <- e
		case <-ctx.Done():
			return
		}
	}
}
