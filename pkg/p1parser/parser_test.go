package p1parser_test

import (
	_ "embed"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/tlwr/smart-meter-exporter/pkg/p1parser"
)

var (
	//go:embed fixtures/example.txt
	example []byte
)

func TestP1parser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "p1parser")
}

var _ = Describe("Parse", func() {
	It("produces a value", func() {
		telegram, err := p1parser.Parse(example)

		Expect(err).NotTo(HaveOccurred())
		Expect(telegram).To(Equal(&p1parser.EnergieTelegram{
			VerbruikTarief1: 13977.847,
			VerbruikTarief2: 014745.839,

			TeruggeleverdTarief1: 0,
			TeruggeleverdTarief2: 0,
			HuidigVerbruik:       0.167,
			HuidigTeruglevering:  0,

			VerbruikTotaal:      28723.686,
			TeruggeleverdTotaal: 0,
		}))
	})
})
