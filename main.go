package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
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

	srl := &serial.Config{
		Name: *serialPath,
		Baud: *baud,
	}
	srlPort, err := serial.OpenPort(srl)
	if err != nil {
		log.Fatalf("kon niet serial device openen: %s", err)
	}

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
					log.Fatal(fmt.Errorf("kon niet van buffer lezen: %w", err))
				}
				if n == 0 {
					continue
				}
				if 600 > n || n > 800 {
					log.Printf("overslaan, waarschijnlijk iets mis met data van %d", n)
					continue
				}
				tg, err := p1parser.Parse(buf)
				if err != nil {
					log.Fatal(fmt.Errorf("kon niet de telegram parsen: %w", err))
				}

				huidigVerbruik.Set(tg.HuidigVerbruik)
				totaalVerbruik.Set(tg.VerbruikTotaal)
			}
		}
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
