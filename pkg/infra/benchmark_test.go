package infra

import (
	"context"
	"testing"

	"tape/e2e/mock"
	"tape/pkg/infra/basic"

	"github.com/hyperledger/fabric-protos-go/peer"
	log "github.com/sirupsen/logrus"
)

func StartProposer(ctx context.Context, signed, processed chan *Elements, logger *log.Logger, threshold int, addr string) {
	peer := basic.Node{
		Addr: addr,
	}
	Proposer, _ := CreateProposer(peer, logger)
	go Proposer.Start(ctx, signed, processed, threshold)
}

func benchmarkNPeer(concurrency int, b *testing.B) {
	processed := make(chan *Elements, 10)
	signeds := make([]chan *Elements, concurrency)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < concurrency; i++ {
		signeds[i] = make(chan *Elements, 10)
		mockpeer, err := mock.NewServer(1, nil)
		if err != nil {
			b.Fatal(err)
		}
		mockpeer.Start()
		defer mockpeer.Stop()
		StartProposer(ctx, signeds[i], processed, nil, concurrency, mockpeer.PeersAddresses()[0])
	}
	b.ReportAllocs()
	b.ResetTimer()
	go func() {
		for i := 0; i < b.N; i++ {
			data := &Elements{SignedProp: &peer.SignedProposal{}}
			for _, s := range signeds {
				s <- data
			}
		}
	}()
	var n int
	for n < b.N {
		<-processed
		n++
	}
	b.StopTimer()
}

func BenchmarkPeerEndorsement1(b *testing.B) { benchmarkNPeer(1, b) }
func BenchmarkPeerEndorsement2(b *testing.B) { benchmarkNPeer(2, b) }
func BenchmarkPeerEndorsement4(b *testing.B) { benchmarkNPeer(4, b) }
func BenchmarkPeerEndorsement8(b *testing.B) { benchmarkNPeer(8, b) }

func benchmarkAsyncCollector(concurrent int, b *testing.B) {
	block := make(chan *AddressedBlock, 100)
	done := make(chan struct{})
	instance, _ := NewBlockCollector(concurrent, concurrent, context.Background(), block, done, b.N, false)
	go instance.Start()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < concurrent; i++ {
		go func(idx int) {
			for j := 0; j < b.N; j++ {
				block <- &AddressedBlock{
					FilteredBlock: &peer.FilteredBlock{
						Number:               uint64(j),
						FilteredTransactions: make([]*peer.FilteredTransaction, 1),
					},
					Address: idx,
				}
			}
		}(i)
	}
	<-done
	b.StopTimer()
}

func BenchmarkAsyncCollector1(b *testing.B)  { benchmarkAsyncCollector(1, b) }
func BenchmarkAsyncCollector2(b *testing.B)  { benchmarkAsyncCollector(2, b) }
func BenchmarkAsyncCollector4(b *testing.B)  { benchmarkAsyncCollector(4, b) }
func BenchmarkAsyncCollector8(b *testing.B)  { benchmarkAsyncCollector(8, b) }
func BenchmarkAsyncCollector16(b *testing.B) { benchmarkAsyncCollector(16, b) }
