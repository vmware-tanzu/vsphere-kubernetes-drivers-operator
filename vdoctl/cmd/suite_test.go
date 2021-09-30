package cmd

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestVDOCtl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "vdoctl test suite")
}
