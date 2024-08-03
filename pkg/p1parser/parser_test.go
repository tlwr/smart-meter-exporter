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

	//go:embed fixtures/example-2.txt
	example2 []byte
)

func TestP1parser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "p1parser")
}

var _ = Describe("Parse", func() {
	It("example 1 produces a value", func() {
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

	It("example 2 produces a value", func() {
		telegram, err := p1parser.Parse(example2)

		Expect(err).NotTo(HaveOccurred())
		Expect(telegram).To(Equal(&p1parser.EnergieTelegram{
			VerbruikTarief1: 2536.701,
			VerbruikTarief2: 1830.239,

			TeruggeleverdTarief1: 406.811,
			TeruggeleverdTarief2: 1032.602,
			HuidigVerbruik:       0,
			HuidigTeruglevering:  1.161,

			VerbruikTotaal:      4366.9400000000005,
			TeruggeleverdTotaal: 1439.413,
		}))
	})
})
