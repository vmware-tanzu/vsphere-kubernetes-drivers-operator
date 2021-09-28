package client

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestDynamicClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dynamic Client Test Suite")
}
