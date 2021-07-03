package infra

import (
	"context"
	"time"

	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Observers struct {
	workers []*Observer
}

type Observer struct {
	index   int
	Address string
	d       peer.Deliver_DeliverFilteredClient
	logger  *log.Logger
}

func CreateObservers(ctx context.Context, channel string, nodes []Node, crypto Crypto, logger *log.Logger) (*Observers, error) {
	var workers []*Observer
	for i, node := range nodes {
		worker, err := CreateObserver(ctx, channel, node, crypto, logger)
		if err != nil {
			return nil, err
		}
		worker.index = i
		workers = append(workers, worker)
	}
	return &Observers{workers: workers}, nil
}

func (o *Observers) Start(errorCh chan error, blockCh chan<- *AddressedBlock, now time.Time) {
	for i := 0; i < len(o.workers); i++ {
		go o.workers[i].Start(errorCh, blockCh, now)
	}
}

func CreateObserver(ctx context.Context, channel string, node Node, crypto Crypto, logger *log.Logger) (*Observer, error) {
	seek, err := CreateSignedDeliverNewestEnv(channel, crypto)
	if err != nil {
		return nil, err
	}

	deliverer, err := CreateDeliverFilteredClient(ctx, node, logger)
	if err != nil {
		return nil, err
	}

	if err = deliverer.Send(seek); err != nil {
		return nil, err
	}

	// drain first response
	if _, err = deliverer.Recv(); err != nil {
		return nil, err
	}

	return &Observer{Address: node.Addr, d: deliverer, logger: logger}, nil
}

func (o *Observer) Start(errorCh chan error, blockCh chan<- *AddressedBlock, now time.Time) {
	o.logger.Debugf("start observer for peer %s", o.Address)

	for {
		r, err := o.d.Recv()
		if err != nil {
			errorCh <- err
		}

		if r == nil {
			errorCh <- errors.Errorf("received nil message, but expect a valid block instead. You could look into your peer logs for more info")
			return
		}

		fb := r.Type.(*peer.DeliverResponse_FilteredBlock)
		o.logger.Debugf("receivedTime %8.2fs\tBlock %6d\tTx %6d\t Address %s\n", time.Since(now).Seconds(), fb.FilteredBlock.Number, len(fb.FilteredBlock.FilteredTransactions), o.Address)

		blockCh <- &AddressedBlock{fb.FilteredBlock, o.index}
	}
}
