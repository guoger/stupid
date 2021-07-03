package basic_test

import (
	"context"
	"tape/pkg/infra/basic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var _ = Describe("Client", func() {

	Context("Client Error handling", func() {
		dummy := basic.Node{
			Addr: "invalid_addr",
		}
		logger := log.New()

		It("captures error from endorser", func() {
			_, err := basic.CreateEndorserClient(dummy, logger)
			Expect(err).Should(MatchError(ContainSubstring("error connecting to invalid_addr")))
		})
		It("captures error from broadcaster", func() {
			_, err := basic.CreateBroadcastClient(context.Background(), dummy, logger)
			Expect(err).Should(MatchError(ContainSubstring("error connecting to invalid_addr")))
		})
		It("captures error from DeliverFilter", func() {
			_, err := basic.CreateDeliverFilteredClient(context.Background(), dummy, logger)
			Expect(err).Should(MatchError(ContainSubstring("error connecting to invalid_addr")))
		})
	})

})
