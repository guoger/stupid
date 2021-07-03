package basic

import (
	"context"
	"crypto/tls"
	"time"

	"tape/internal/fabric/core/comm"

	"github.com/hyperledger/fabric-protos-go/orderer"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func CreateGRPCClient(node Node) (*comm.GRPCClient, error) {
	var certs [][]byte
	if node.TLSCACertByte != nil {
		certs = append(certs, node.TLSCACertByte)
	}
	config := comm.ClientConfig{}
	config.Timeout = 5 * time.Second
	config.SecOpts = comm.SecureOptions{
		UseTLS:            false,
		RequireClientCert: false,
		ServerRootCAs:     certs,
	}

	if len(certs) > 0 {
		config.SecOpts.UseTLS = true
		if len(node.TLSCAKey) > 0 && len(node.TLSCARoot) > 0 {
			config.SecOpts.RequireClientCert = true
			config.SecOpts.Certificate = node.TLSCACertByte
			config.SecOpts.Key = node.TLSCAKeyByte
			if node.TLSCARootByte != nil {
				config.SecOpts.ClientRootCAs = append(config.SecOpts.ClientRootCAs, node.TLSCARootByte)
			}
		}
	}

	grpcClient, err := comm.NewGRPCClient(config)
	//to do: unit test for this error, current fails to make case for this
	if err != nil {
		return nil, errors.Wrapf(err, "error connecting to %s", node.Addr)
	}

	return grpcClient, nil
}

func CreateEndorserClient(node Node, logger *log.Logger) (peer.EndorserClient, error) {
	conn, err := DailConnection(node, logger)
	if err != nil {
		return nil, err
	}
	return peer.NewEndorserClient(conn), nil
}

func CreateBroadcastClient(ctx context.Context, node Node, logger *log.Logger) (orderer.AtomicBroadcast_BroadcastClient, error) {
	conn, err := DailConnection(node, logger)
	if err != nil {
		return nil, err
	}
	return orderer.NewAtomicBroadcastClient(conn).Broadcast(ctx)
}

func CreateDeliverFilteredClient(ctx context.Context, node Node, logger *log.Logger) (peer.Deliver_DeliverFilteredClient, error) {
	conn, err := DailConnection(node, logger)
	if err != nil {
		return nil, err
	}
	return peer.NewDeliverClient(conn).DeliverFiltered(ctx)
}

// TODO: use a global get logger function instead inject a logger
func DailConnection(node Node, logger *log.Logger) (*grpc.ClientConn, error) {
	gRPCClient, err := CreateGRPCClient(node)
	if err != nil {
		return nil, err
	}
	var connError error
	var conn *grpc.ClientConn
	for i := 1; i <= 3; i++ {
		conn, connError = gRPCClient.NewConnection(node.Addr, func(tlsConfig *tls.Config) { tlsConfig.InsecureSkipVerify = true })
		if connError == nil {
			return conn, nil
		} else {
			logger.Errorf("%d of 3 attempts to make connection to %s, details: %s", i, node.Addr, connError)
		}
	}
	return nil, errors.Wrapf(connError, "error connecting to %s", node.Addr)
}
