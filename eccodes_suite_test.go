package eccodes

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEccodes(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ecCodes Suite")
}
