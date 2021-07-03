package infra

import (
	"context"

	"github.com/pkg/errors"
	"golang.org/x/time/rate"
)

func StartCreateProposal(num int, burst int, r float64, config Config, crypto Crypto, raw chan *Elements, errorCh chan error) {
	limit := rate.Inf
	ctx := context.Background()
	if r > 0 {
		limit = rate.Limit(r)
	}
	limiter := rate.NewLimiter(limit, burst)
	for i := 0; i < num; i++ {
		prop, err := CreateProposal(
			crypto,
			config.Channel,
			config.Chaincode,
			config.Version,
			config.Args...,
		)
		if err != nil {
			errorCh <- errors.Wrapf(err, "error creating proposal")
			return
		}

		if err = limiter.Wait(ctx); err != nil {
			errorCh <- errors.Wrapf(err, "error creating proposal")
			return
		}

		raw <- &Elements{Proposal: prop}
	}
}
