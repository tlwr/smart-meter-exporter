package p1parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

var (
	verbruikTarief1Prefix = []byte("1.8.1")
	verbruikTarief2Prefix = []byte("1.8.2")

	terugleveringTarief1Prefix = []byte("2.8.1")
	terugleveringTarief2Prefix = []byte("2.8.2")

	huidigVerbruikPrefix      = []byte("1.7.0")
	huidigTerugleveringPrefix = []byte("2.7.0")

	unitKiloWattUrenExtractor = regexp.MustCompile("(?<Powert>(([0-9]*)([.][0-9]+)?)(?:[*]kWh))")
	unitKiloWattExtractor     = regexp.MustCompile("(?<Powert>(([0-9]*)([.][0-9]+)?)(?:[*]kW))")
)

type EnergieTelegram struct {
	VerbruikTarief1 float64
	VerbruikTarief2 float64
	VerbruikTotaal  float64

	TeruggeleverdTarief1 float64
	TeruggeleverdTarief2 float64
	TeruggeleverdTotaal  float64

	HuidigVerbruik      float64
	HuidigTeruglevering float64
}

func Parse(b []byte) (*EnergieTelegram, error) {
	tg := &EnergieTelegram{}

	errs := []error{}
	s := bufio.NewScanner(bytes.NewReader(b))

	for s.Scan() {
		line := s.Bytes()
		line = bytes.ReplaceAll(line, []byte("\x00"), []byte(""))

		if len(line) < 10 {
			continue
		}

		line = line[4:]

		if bytes.HasPrefix(line, verbruikTarief1Prefix) {
			kiloWattUrenParsen(line, &tg.VerbruikTarief1, &errs)
		} else if bytes.HasPrefix(line, verbruikTarief2Prefix) {
			kiloWattUrenParsen(line, &tg.VerbruikTarief2, &errs)
		} else if bytes.HasPrefix(line, terugleveringTarief1Prefix) {
			kiloWattUrenParsen(line, &tg.TeruggeleverdTarief1, &errs)
		} else if bytes.HasPrefix(line, terugleveringTarief2Prefix) {
			kiloWattUrenParsen(line, &tg.TeruggeleverdTarief2, &errs)
		} else if bytes.HasPrefix(line, huidigVerbruikPrefix) {
			kiloWattParsen(line, &tg.HuidigVerbruik, &errs)
		} else if bytes.HasPrefix(line, huidigTerugleveringPrefix) {
			kiloWattParsen(line, &tg.HuidigTeruglevering, &errs)
		}
	}

	tg.VerbruikTotaal = tg.VerbruikTarief1 + tg.VerbruikTarief2
	tg.TeruggeleverdTotaal = tg.TeruggeleverdTarief1 + tg.TeruggeleverdTarief2

	return tg, errors.Join(errs...)
}

func kiloWattUrenParsen(stuk []byte, dest *float64, errAcc *[]error) {
	stukZonderPrefix := stuk[5:]

	match := unitKiloWattUrenExtractor.FindSubmatch(stukZonderPrefix)
	if len(match) != 5 {
		*errAcc = append(*errAcc, fmt.Errorf("vond rare match %q (%d) ", string(stuk), len(match)))
		return
	}

	val, err := strconv.ParseFloat(string(match[2]), 64)
	if err != nil {
		*errAcc = append(*errAcc, fmt.Errorf("kon niet parsen %q: %w", string(match[1]), err))
		return
	}

	*dest = val
}

func kiloWattParsen(stuk []byte, dest *float64, errAcc *[]error) {
	stukZonderPrefix := stuk[5:]

	match := unitKiloWattExtractor.FindSubmatch(stukZonderPrefix)
	if len(match) != 5 {
		*errAcc = append(*errAcc, fmt.Errorf("vond rare match %q (%d) ", string(stuk), len(match)))
		return
	}

	val, err := strconv.ParseFloat(string(match[2]), 64)
	if err != nil {
		*errAcc = append(*errAcc, fmt.Errorf("kon niet parsen %q: %w", string(match[1]), err))
		return
	}

	*dest = val
}
