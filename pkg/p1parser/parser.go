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

	actieveStroomPositiefP1 = []byte("21.7.0")
	actieveStroomPositiefP2 = []byte("41.7.0")
	actieveStroomPositiefP3 = []byte("61.7.0")
	actieveStroomNegatiefP1 = []byte("22.7.0")
	actieveStroomNegatiefP2 = []byte("42.7.0")
	actieveStroomNegatiefP3 = []byte("62.7.0")

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

	ActieveStroomPositiefP1 float64
	ActieveStroomPositiefP2 float64
	ActieveStroomPositiefP3 float64

	ActieveStroomNegatiefP1 float64
	ActieveStroomNegatiefP2 float64
	ActieveStroomNegatiefP3 float64
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
		} else if bytes.HasPrefix(line, actieveStroomPositiefP1) {
			kiloWattParsen(line, &tg.ActieveStroomPositiefP1, &errs)
		} else if bytes.HasPrefix(line, actieveStroomPositiefP2) {
			kiloWattParsen(line, &tg.ActieveStroomPositiefP2, &errs)
		} else if bytes.HasPrefix(line, actieveStroomPositiefP3) {
			kiloWattParsen(line, &tg.ActieveStroomPositiefP3, &errs)
		} else if bytes.HasPrefix(line, actieveStroomNegatiefP1) {
			kiloWattParsen(line, &tg.ActieveStroomNegatiefP1, &errs)
		} else if bytes.HasPrefix(line, actieveStroomNegatiefP2) {
			kiloWattParsen(line, &tg.ActieveStroomNegatiefP2, &errs)
		} else if bytes.HasPrefix(line, actieveStroomNegatiefP3) {
			kiloWattParsen(line, &tg.ActieveStroomNegatiefP3, &errs)
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
