package basic_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBasic(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Basic Suite")
}
