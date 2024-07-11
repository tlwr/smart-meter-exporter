package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tarm/serial"
	"go.opentelemetry.io/otel/exporters/prometheus"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/tlwr/smart-meter-otel-metrics/pkg/p1parser"
)

var (
	serialPath = flag.String("serial-path", "/dev/ttyUSB0", "path to serial device for p1-over-usb port")
	baud       = flag.Int("baud", 115200, "baud of serial device")
	interval   = flag.Duration("interval", 10*time.Second, "how long to wait between serial reads")
)

func main() {
	flag.Parse()

	rootCtx := context.Background()
	sigCtx, sigStop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer sigStop()

	log.Println("starting otel")
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatalf("kon niet prometheus starten: %s", err)
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	meter := provider.Meter("smart-meter-otel-metrics")

	huidigVerbruik, err := meter.Float64Gauge(
		"verbruik.huidig",
		otelmetric.WithUnit("kW"),
	)
	if err != nil {
		log.Fatalf("kon niet de huidig verbruik metric maken: %s", err)
	}

	totaalVerbruik, err := meter.Float64Gauge(
		"totaal.verbruik",
		otelmetric.WithUnit("kWh"),
	)
	if err != nil {
		log.Fatalf("kon niet de totaal verbruik metric maken: %s", err)
	}

	log.Println("starting serial")
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
				if 700 > n || n > 800 {
					log.Printf("overslaan, waarschijnlijk iets mis met data van %d", n)
					continue
				}
				tg, err := p1parser.Parse(buf)
				if err != nil {
					log.Fatal(fmt.Errorf("kon niet de telegram parsen: %w", err))
				}

				huidigVerbruik.Record(serialCtx, tg.HuidigVerbruik)
				totaalVerbruik.Record(serialCtx, tg.VerbruikTotaal)
			}
		}
	}()

	wg.Add(1)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9090", nil) //nolint:gosec // Ignoring G114: Use of net/http serve function that has no support for setting timeouts.
		if err != nil {
			fmt.Printf("error serving http: %v", err)
			return
		}
	}()

	<-sigCtx.Done()
	sigStop()

	cancelSerial()
	wg.Wait()
	log.Println("stopping")
}
