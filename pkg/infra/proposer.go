package infra

import (
	"context"
	"io"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/orderer"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Proposers struct {
	workers   [][]*Proposer
	logger    *log.Logger
	ctx       context.Context
	signed    []chan *Elements
	processed chan *Elements
	config    Config
}

func CreateProposers(ctx context.Context, signed []chan *Elements, processed chan *Elements, config Config, logger *log.Logger) (*Proposers, error) {
	var ps [][]*Proposer
	var err error
	//one proposer per connection per peer
	for _, node := range config.Endorsers {
		row := make([]*Proposer, config.NumOfConn)
		for j := 0; j < config.NumOfConn; j++ {
			row[j], err = CreateProposer(node, logger)
			if err != nil {
				return nil, err
			}
		}
		ps = append(ps, row)
	}

	return &Proposers{workers: ps, logger: logger, ctx: ctx, signed: signed, processed: processed, config: config}, nil
}

func (ps *Proposers) Start() {
	ps.logger.Infof("Start sending transactions.")
	for i := 0; i < len(ps.config.Endorsers); i++ {
		// peer connection should be config.ClientPerConn * config.NumOfConn
		for k := 0; k < ps.config.ClientPerConn; k++ {
			for j := 0; j < ps.config.NumOfConn; j++ {
				go ps.workers[i][j].Start(ps.ctx, ps.signed[i], ps.processed, len(ps.config.Endorsers))
			}
		}
	}
}

type Proposer struct {
	e      peer.EndorserClient
	Addr   string
	logger *log.Logger
}

func CreateProposer(node Node, logger *log.Logger) (*Proposer, error) {
	endorser, err := CreateEndorserClient(node, logger)
	if err != nil {
		return nil, err
	}
	return &Proposer{e: endorser, Addr: node.Addr, logger: logger}, nil
}

func (p *Proposer) Start(ctx context.Context, signed, processed chan *Elements, threshold int) {
	for {
		select {
		case s := <-signed:
			//send sign proposal to peer for endorsement
			r, err := p.e.ProcessProposal(ctx, s.SignedProp)
			if err != nil || r.Response.Status < 200 || r.Response.Status >= 400 {
				if r == nil {
					p.logger.Errorf("Err processing proposal: %s, status: unknown, addr: %s \n", err, p.Addr)
				} else {
					p.logger.Errorf("Err processing proposal: %s, status: %d, message: %s, addr: %s \n", err, r.Response.Status, r.Response.Message, p.Addr)
				}
				continue
			}
			s.lock.Lock()
			//collect for endorsement
			s.Responses = append(s.Responses, r)
			if len(s.Responses) >= threshold {
				processed <- s
			}
			s.lock.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

type Broadcasters struct {
	workers []*Broadcaster
	ctx     context.Context
	envs    chan *Elements
	errorCh chan error
}

type Broadcaster struct {
	c      orderer.AtomicBroadcast_BroadcastClient
	logger *log.Logger
}

func CreateBroadcasters(ctx context.Context, envs chan *Elements, errorCh chan error, config Config, logger *log.Logger) (*Broadcasters, error) {
	var workers []*Broadcaster
	for i := 0; i < config.NumOfConn; i++ {
		broadcaster, err := CreateBroadcaster(ctx, config.Orderer, logger)
		if err != nil {
			return nil, err
		}
		workers = append(workers, broadcaster)
	}

	return &Broadcasters{
		workers: workers,
		ctx:     ctx,
		envs:    envs,
		errorCh: errorCh,
	}, nil
}

func (bs Broadcasters) Start() {
	for _, b := range bs.workers {
		go b.StartDraining(bs.errorCh)
		go b.Start(bs.ctx, bs.envs, bs.errorCh)
	}
}

func CreateBroadcaster(ctx context.Context, node Node, logger *log.Logger) (*Broadcaster, error) {
	client, err := CreateBroadcastClient(ctx, node, logger)
	if err != nil {
		return nil, err
	}

	return &Broadcaster{c: client, logger: logger}, nil
}

func (b *Broadcaster) Start(ctx context.Context, envs <-chan *Elements, errorCh chan error) {
	b.logger.Debugf("Start sending broadcast")
	for {
		select {
		case e := <-envs:
			err := b.c.Send(e.Envelope)
			if err != nil {
				errorCh <- err
			}
		case <-ctx.Done():
			return
		}
	}
}

func (b *Broadcaster) StartDraining(errorCh chan error) {
	for {
		res, err := b.c.Recv()
		if err != nil {
			if err == io.EOF {
				return
			}
			b.logger.Errorf("recv broadcast err: %+v, status: %+v\n", err, res)
			return
		}

		if res.Status != common.Status_SUCCESS {
			errorCh <- errors.Errorf("recv errouneous status %s", res.Status)
			return
		}
	}
}
