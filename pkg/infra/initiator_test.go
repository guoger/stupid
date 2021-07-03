package infra_test

import (
	"io/ioutil"
	"os"
	"time"

	"tape/e2e"
	"tape/pkg/infra"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Initiator", func() {

	var (
		configFile *os.File
		tmpDir     string
	)

	BeforeEach(func() {
		tmpDir, err := ioutil.TempDir("", "tape-")
		Expect(err).NotTo(HaveOccurred())

		mtlsCertFile, err := ioutil.TempFile(tmpDir, "mtls-*.crt")
		Expect(err).NotTo(HaveOccurred())

		mtlsKeyFile, err := ioutil.TempFile(tmpDir, "mtls-*.key")
		Expect(err).NotTo(HaveOccurred())

		err = e2e.GenerateCertAndKeys(mtlsKeyFile, mtlsCertFile)
		Expect(err).NotTo(HaveOccurred())

		mtlsCertFile.Close()
		mtlsKeyFile.Close()

		configFile, err = ioutil.TempFile(tmpDir, "config*.yaml")
		Expect(err).NotTo(HaveOccurred())
		configValue := e2e.Values{
			PrivSk:          mtlsKeyFile.Name(),
			SignCert:        mtlsCertFile.Name(),
			Mtls:            false,
			PeersAddrs:      []string{"dummy"},
			OrdererAddr:     "dummy",
			CommitThreshold: 1,
		}
		e2e.GenerateConfigFile(configFile.Name(), configValue)
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("should crete proposal to raw without limit when limit is 0", func() {

		raw := make(chan *infra.Elements, 1002)
		defer close(raw)
		errorCh := make(chan error, 1002)
		defer close(errorCh)
		config, err := infra.LoadConfig(configFile.Name())
		Expect(err).NotTo(HaveOccurred())
		crypto, err := config.LoadCrypto()
		Expect(err).NotTo(HaveOccurred())
		t := time.Now()
		Initiator := &infra.Initiator{1002, 10, 0, config, crypto, raw, errorCh}
		Initiator.Start()
		t1 := time.Now()
		Expect(raw).To(HaveLen(1002))
		Expect(t1.Sub(t)).To(BeNumerically("<", 2*time.Second))

	})

	It("should crete proposal to raw with given limit bigger than 0 less than size", func() {
		raw := make(chan *infra.Elements, 1002)
		defer close(raw)
		errorCh := make(chan error, 1002)
		defer close(errorCh)
		config, err := infra.LoadConfig(configFile.Name())
		Expect(err).NotTo(HaveOccurred())
		crypto, err := config.LoadCrypto()
		Expect(err).NotTo(HaveOccurred())
		t := time.Now()
		Initiator := &infra.Initiator{12, 10, 1, config, crypto, raw, errorCh}
		Initiator.Start()
		t1 := time.Now()
		Expect(raw).To(HaveLen(12))
		Expect(t1.Sub(t)).To(BeNumerically(">", 2*time.Second))
	})

	It("should crete proposal to raw with given limit bigger than Size", func() {
		raw := make(chan *infra.Elements, 1002)
		defer close(raw)
		errorCh := make(chan error, 1002)
		defer close(errorCh)
		config, err := infra.LoadConfig(configFile.Name())
		Expect(err).NotTo(HaveOccurred())
		crypto, err := config.LoadCrypto()
		Expect(err).NotTo(HaveOccurred())
		t := time.Now()
		Initiator := &infra.Initiator{12, 10, 0, config, crypto, raw, errorCh}
		Initiator.Start()
		t1 := time.Now()
		Expect(raw).To(HaveLen(12))
		Expect(t1.Sub(t)).To(BeNumerically("<", 2*time.Second))
	})
})
