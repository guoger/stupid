package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Process(configPath string, num int, burst int, rate float64, logger *log.Logger) error {
	/*** variables ***/
	config, err := LoadConfig(configPath)
	if err != nil {
		return err
	}
	crypto, err := config.LoadCrypto()
	if err != nil {
		return err
	}
	raw := make(chan *Elements, burst)
	signed := make([]chan *Elements, len(config.Endorsers))
	processed := make(chan *Elements, burst)
	envs := make(chan *Elements, burst)
	blockCh := make(chan *AddressedBlock)
	finishCh := make(chan struct{})
	errorCh := make(chan error, burst)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < len(config.Endorsers); i++ {
		signed[i] = make(chan *Elements, burst)
	}
	/*** workers ***/

	blockCollector, err := NewBlockCollector(config.CommitThreshold, len(config.Committers), ctx, blockCh, finishCh, num, true)
	if err != nil {
		return errors.Wrap(err, "failed to create block collector")
	}
	Initiator := &Initiator{num, burst, rate, config, crypto, raw, errorCh}
	assembler := &Assembler{crypto, ctx, raw, signed, errorCh}
	proposers, err := CreateProposers(ctx, signed, processed, config, logger)
	if err != nil {
		return err
	}
	Integrator := &Integrator{crypto, ctx, processed, envs, errorCh}
	broadcaster, err := CreateBroadcasters(ctx, envs, errorCh, config, logger)
	if err != nil {
		return err
	}

	observers, err := CreateObservers(ctx, crypto, errorCh, blockCh, config, logger)
	if err != nil {
		return err
	}
	/*** start workers ***/

	proposers.Start()
	broadcaster.Start()

	go blockCollector.Start()
	go observers.Start()

	for i := 0; i < 5; i++ {
		go assembler.Start()
		go Integrator.Start()
	}

	go Initiator.Start()
	/*** waiting for complete ***/
	for {
		select {
		case err = <-errorCh:
			return err
		case <-finishCh:
			duration := time.Since(observers.StartTime)
			logger.Infof("Completed processing transactions.")
			fmt.Printf("tx: %d, duration: %+v, tps: %f\n", num, duration, float64(num)/duration.Seconds())
			return nil
		}
	}
}
