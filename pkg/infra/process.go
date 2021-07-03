package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Process(configPath string, num int, burst int, rate float64, logger *log.Logger) error {
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
	/*****************/
	assembler := &Assembler{crypto, ctx, raw, signed, errorCh}
	Integrator := &Integrator{crypto, ctx, processed, envs, errorCh}
	Initiator := &Initiator{num, burst, rate, config, crypto, raw, errorCh}
	/*****************/

	blockCollector, err := NewBlockCollector(config.CommitThreshold, len(config.Committers))
	if err != nil {
		return errors.Wrap(err, "failed to create block collector")
	}
	for i := 0; i < len(config.Endorsers); i++ {
		signed[i] = make(chan *Elements, burst)
	}

	proposers, err := CreateProposers(config.NumOfConn, config.Endorsers, logger)
	if err != nil {
		return err
	}
	proposers.Start(ctx, signed, processed, config)

	broadcaster, err := CreateBroadcasters(ctx, config.NumOfConn, config.Orderer, logger)
	if err != nil {
		return err
	}
	broadcaster.Start(ctx, envs, errorCh)

	observers, err := CreateObservers(ctx, config.Channel, config.Committers, crypto, logger)
	if err != nil {
		return err
	}

	start := time.Now()

	go blockCollector.Start(ctx, blockCh, finishCh, num, time.Now(), true)
	go observers.Start(errorCh, blockCh, start)
	for i := 0; i < 5; i++ {
		go assembler.Start()
		go Integrator.Start()
	}
	go Initiator.Start()
	/*****************/
	/*Waiting for complete*/
	/*****************/
	for {
		select {
		case err = <-errorCh:
			return err
		case <-finishCh:
			duration := time.Since(start)
			logger.Infof("Completed processing transactions.")
			fmt.Printf("tx: %d, duration: %+v, tps: %f\n", num, duration, float64(num)/duration.Seconds())
			return nil
		}
	}
}
