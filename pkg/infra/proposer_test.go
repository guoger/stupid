package infra_test

import (
	"context"
	"tape/e2e/mock"
	"tape/pkg/infra"
	"tape/pkg/infra/basic"

	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("Proposer", func() {

	var addr string
	var logger = log.New()
	var processed chan *infra.Elements

	BeforeEach(func() {
		srv := &mocks.MockEndorserServer{}
		addr = srv.Start("127.0.0.1:0")
	})

	Context("CreateProposer", func() {
		It("successfully creates a proposer", func() {
			dummy := basic.Node{
				Addr: addr,
			}
			Proposer, err := infra.CreateProposer(dummy, logger)
			Expect(Proposer.Addr).To(Equal(addr))
			Expect(err).NotTo(HaveOccurred())
		})

		It("handle error ", func() {
			dummy := basic.Node{
				Addr: "invalid_addr",
			}
			_, err := infra.CreateProposer(dummy, logger)
			Expect(err).Should(MatchError(ContainSubstring("error connecting to invalid_addr")))
		})
	})

	Context("CreateBroadcasters", func() {
		It("successfully creates a broadcaster", func() {
			dummy := basic.Node{
				Addr: addr,
			}
			Broadcaster, err := infra.CreateBroadcaster(context.Background(), dummy, logger)
			Expect(Broadcaster).NotTo(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})

		It("captures connection errors", func() {
			dummy := basic.Node{
				Addr: "invalid_addr",
			}
			_, err := infra.CreateBroadcaster(context.Background(), dummy, logger)
			Expect(err).Should(MatchError(ContainSubstring("error connecting to invalid_addr")))
		})
	})

	Context("Tape should do less for prepare and summary endorsement process", func() {
		// 0.002 here for mac testing on azp
		// For ginkgo,
		// You may only call Measure from within a Describe, Context or When
		// So here only tested with concurrent as 8 peers
		Measure("it should do endorsement efficiently for 2 peers", func(b Benchmarker) {
			peerNum := 2
			processed = make(chan *infra.Elements, 10)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			signeds := make([]chan *infra.Elements, peerNum)
			for i := 0; i < peerNum; i++ {
				signeds[i] = make(chan *infra.Elements, 10)
				mockpeer, err := mock.NewServer(1, nil)
				Expect(err).NotTo(HaveOccurred())
				mockpeer.Start()
				infra.StartProposer(ctx, signeds[i], processed, nil, peerNum, mockpeer.PeersAddresses()[0])
				defer mockpeer.Stop()
			}
			runtime := b.Time("runtime", func() {
				data := &infra.Elements{SignedProp: &peer.SignedProposal{}}
				for _, s := range signeds {
					s <- data
				}
				<-processed
			})
			Expect(runtime.Seconds()).Should(BeNumerically("<", 0.002), "endorsement() shouldn't take too long.")
		}, 10)
	})
})
