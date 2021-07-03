package infra_test

import (
	"context"
	"sync"
	"tape/pkg/infra"

	"github.com/hyperledger/fabric-protos-go/peer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func newAddressedBlock(addr int, blockNum uint64) *infra.AddressedBlock {
	return &infra.AddressedBlock{Address: addr, FilteredBlock: &peer.FilteredBlock{Number: blockNum, FilteredTransactions: make([]*peer.FilteredTransaction, 1)}}
}

var _ = Describe("BlockCollector", func() {

	Context("Async Commit", func() {
		It("should work with threshold 1 and observer 1", func() {
			block := make(chan *infra.AddressedBlock)
			done := make(chan struct{})
			instance, err := infra.NewBlockCollector(1, 1, context.Background(), block, done, 2, false)
			Expect(err).NotTo(HaveOccurred())

			go instance.Start()

			block <- newAddressedBlock(0, 0)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(0, 1)
			Eventually(done).Should(BeClosed())
		})

		It("should work with threshold 1 and observer 2", func() {
			block := make(chan *infra.AddressedBlock)
			done := make(chan struct{})
			instance, err := infra.NewBlockCollector(1, 2, context.Background(), block, done, 2, false)
			Expect(err).NotTo(HaveOccurred())

			go instance.Start()

			block <- newAddressedBlock(0, 0)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(1, 0)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(0, 1)
			Eventually(done).Should(BeClosed())

			select {
			case block <- newAddressedBlock(1, 1):
			default:
				Fail("Block collector should still be able to consume blocks")
			}
		})

		It("should work with threshold 4 and observer 4", func() {
			block := make(chan *infra.AddressedBlock)
			done := make(chan struct{})
			instance, err := infra.NewBlockCollector(4, 4, context.Background(), block, done, 2, false)
			Expect(err).NotTo(HaveOccurred())

			go instance.Start()

			block <- newAddressedBlock(0, 1)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(1, 1)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(2, 1)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(3, 1)
			Consistently(done).ShouldNot(BeClosed())

			block <- newAddressedBlock(0, 0)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(1, 0)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(2, 0)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(3, 0)
			Eventually(done).Should(BeClosed())
		})

		It("should work with threshold 2 and observer 4", func() {
			block := make(chan *infra.AddressedBlock)
			done := make(chan struct{})
			instance, err := infra.NewBlockCollector(2, 4, context.Background(), block, done, 1, false)
			Expect(err).NotTo(HaveOccurred())

			go instance.Start()

			block <- newAddressedBlock(0, 0)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(1, 0)
			Eventually(done).Should(BeClosed())
		})

		PIt("should not count tx for repeated block", func() {
			block := make(chan *infra.AddressedBlock)
			done := make(chan struct{})
			instance, err := infra.NewBlockCollector(1, 1, context.Background(), block, done, 2, false)
			Expect(err).NotTo(HaveOccurred())

			go instance.Start()

			block <- newAddressedBlock(0, 0)
			Consistently(done).ShouldNot(BeClosed())
			block <- newAddressedBlock(0, 0)
			Consistently(done).ShouldNot(BeClosed())

			block <- newAddressedBlock(0, 1)
			Eventually(done).Should(BeClosed())
		})

		It("should return err when threshold is greater than total", func() {
			block := make(chan *infra.AddressedBlock)
			done := make(chan struct{})
			instance, err := infra.NewBlockCollector(2, 1, context.Background(), block, done, 2, false)
			Expect(err).Should(MatchError("threshold [2] must be less than or equal to total [1]"))
			Expect(instance).Should(BeNil())
		})

		It("should return err when threshold or total is zero", func() {
			block := make(chan *infra.AddressedBlock)
			done := make(chan struct{})
			instance, err := infra.NewBlockCollector(0, 1, context.Background(), block, done, 2, false)
			Expect(err).Should(MatchError("threshold and total must be greater than zero"))
			Expect(instance).Should(BeNil())

			instance, err = infra.NewBlockCollector(1, 0, context.Background(), block, done, 2, false)
			Expect(err).Should(MatchError("threshold and total must be greater than zero"))
			Expect(instance).Should(BeNil())
		})

		It("Should supports parallel committers", func() {
			block := make(chan *infra.AddressedBlock)
			done := make(chan struct{})
			instance, err := infra.NewBlockCollector(100, 100, context.Background(), block, done, 1, false)
			Expect(err).NotTo(HaveOccurred())

			go instance.Start()

			var wg sync.WaitGroup
			wg.Add(100)
			for i := 0; i < 100; i++ {
				go func(idx int) {
					defer wg.Done()
					block <- newAddressedBlock(idx, 0)
				}(i)
			}
			wg.Wait()
			Eventually(done).Should(BeClosed())
		})

		It("Should supports threshold 3 and observer 5 as parallel committers", func() {
			block := make(chan *infra.AddressedBlock)
			done := make(chan struct{})
			instance, err := infra.NewBlockCollector(3, 5, context.Background(), block, done, 10, false)
			Expect(err).NotTo(HaveOccurred())

			go instance.Start()

			for i := 0; i < 3; i++ {
				go func(idx int) {
					for j := 0; j < 10; j++ {
						block <- newAddressedBlock(idx, uint64(j))
					}
				}(i)
			}
			Eventually(done).Should(BeClosed())
		})

		It("Should supports threshold 5 and observer 5 as parallel committers", func() {
			block := make(chan *infra.AddressedBlock)
			done := make(chan struct{})
			instance, err := infra.NewBlockCollector(5, 5, context.Background(), block, done, 10, false)
			Expect(err).NotTo(HaveOccurred())

			go instance.Start()
			for i := 0; i < 5; i++ {
				go func(idx int) {
					for j := 0; j < 10; j++ {
						block <- newAddressedBlock(idx, uint64(j))
					}
				}(i)
			}
			Eventually(done).Should(BeClosed())
		})
	})
})
