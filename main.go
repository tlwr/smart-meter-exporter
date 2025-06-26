package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tarm/serial"

	"github.com/tlwr/smart-meter-exporter/pkg/p1parser"
	"github.com/tlwr/smart-meter-exporter/pkg/p1telegramcollector"
)

var (
	serialPath = flag.String("serial-path", "/dev/ttyUSB0", "path to serial device for p1-over-usb port")
	baud       = flag.Int("baud", 115200, "baud of serial device")
	interval   = flag.Duration("interval", 10*time.Second, "how long to wait between serial reads")
	promAddr   = flag.String("prometheus-addr", ":9220", "listen address for prometheus metrics")
)

func main() {
	flag.Parse()

	rootCtx := context.Background()
	sigCtx, sigStop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer sigStop()

	registry := prometheus.NewRegistry()

	huidigVerbruik := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "huidig_verbruik_kw",
		Help: "Hoeveel stroom is nu gebruikt te zijn",
	})
	registry.MustRegister(huidigVerbruik)

	totaalVerbruik := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "totaal_verbruik_kwh",
		Help: "Hoeveel stroomverbruik is gemeten door de meter",
	})
	registry.MustRegister(totaalVerbruik)

	huidigTeruglevering := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "huidig_teruglevering_kw",
		Help: "Hoeveel stroom is nu teruggeleverd te zijn",
	})
	registry.MustRegister(huidigTeruglevering)

	totalTeruglevering := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "totaal_teruglevering_kwh",
		Help: "Hoeveel teruglevering is gemeten door de meter",
	})
	registry.MustRegister(totalTeruglevering)

	actieveStroomP1 := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "actieve_stroom_p1_kw",
		Help: "Huidige stroom van het laatste meetpunt P1",
	})
	registry.MustRegister(actieveStroomP1)

	actieveStroomP2 := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "actieve_stroom_p2_kw",
		Help: "Huidige stroom van het laatste meetpunt P3",
	})
	registry.MustRegister(actieveStroomP2)

	actieveStroomP3 := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "actieve_stroom_p3_kw",
		Help: "Huidige stroom van het laatste meetpunt P3",
	})
	registry.MustRegister(actieveStroomP3)

	srl := &serial.Config{
		Name: *serialPath,
		Baud: *baud,
	}
	srlPort, err := serial.OpenPort(srl)
	if err != nil {
		log.Fatalf("kon niet serial device openen: %s", err)
	}

	collCtx, cancelColl := context.WithCancel(rootCtx)
	coll := p1telegramcollector.New()
	coll.Run(collCtx)

	wg := sync.WaitGroup{}

	serialCtx, cancelSerial := context.WithCancel(sigCtx)
	wg.Add(1)
	go func() {
		log.Println("van de serial aan het lezen")
		for {
			time.Sleep(*interval)

			select {
			case <-serialCtx.Done():
				log.Println("klaar met lezen van serial")
				wg.Done()
				return
			default:
				buf := make([]byte, 1024)
				n, err := srlPort.Read(buf)
				if err != nil {
					log.Printf("kon niet van buffer lezen: %s", err)
					continue
				}
				if n == 0 {
					continue
				}

				coll.Write(buf)
			}
		}
	}()

	wg.Add(1)
	go func() {
		for telegram := range coll.C {
			if telegram == nil {
				break
			}

			tg, err := p1parser.Parse(telegram.Bytes())
			if err != nil {
				log.Printf("kon niet de telegram parsen: %s", err)
				continue
			}

			log.Printf("telegram %v", tg)

			huidigVerbruik.Set(tg.HuidigVerbruik)
			totaalVerbruik.Set(tg.VerbruikTotaal)
			huidigTeruglevering.Set(tg.HuidigTeruglevering)
			totalTeruglevering.Set(tg.TeruggeleverdTotaal)

			if tg.ActieveStroomPositiefP1 > 0.0 {
				actieveStroomP1.Set(tg.ActieveStroomPositiefP1)
			} else {
				actieveStroomP1.Set(-1 * tg.ActieveStroomNegatiefP1)
			}

			if tg.ActieveStroomPositiefP2 > 0.0 {
				actieveStroomP2.Set(tg.ActieveStroomPositiefP2)
			} else {
				actieveStroomP2.Set(-1 * tg.ActieveStroomNegatiefP2)
			}

			if tg.ActieveStroomPositiefP3 > 0.0 {
				actieveStroomP3.Set(tg.ActieveStroomPositiefP3)
			} else {
				actieveStroomP3.Set(-1 * tg.ActieveStroomNegatiefP3)
			}
		}

		wg.Done()
	}()

	promServer := http.Server{
		Addr: *promAddr,
		Handler: promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{
				EnableOpenMetrics: false,
			}),
	}

	wg.Add(1)
	go func() {
		log.Printf("prometheus starten: %s", *promAddr)
		err := promServer.ListenAndServe() //nolint:gosec // Ignoring G114: Use of net/http serve function that has no support for setting timeouts.
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("error prometheus server: %v", err)
			return
		}
		wg.Done()
	}()

	<-sigCtx.Done()
	sigStop()
	log.Println("signaal te stoppen ontvangen")

	cancelColl()
	cancelSerial()

	psCtx, psCancel := context.WithTimeout(rootCtx, 15*time.Second)
	defer psCancel()

	if err := promServer.Shutdown(psCtx); err != nil {
		log.Fatalf("kon niet prometheus server uitschakkelen: %s", err)
	} else {
		log.Println("prometheus gestopd")
	}
	psCancel()

	wg.Wait()
	log.Println("exit")
}
