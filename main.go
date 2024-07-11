package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/tarm/serial"

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

	log.Println("starting serial")
	srl := &serial.Config{
		Name: *serialPath,
		Baud: *baud,
	}
	srlPort, err := serial.OpenPort(srl)
	if err != nil {
		log.Fatalf(fmt.Sprintf("kon niet serial device openen: %s", err))
	}

	serialCtx, cancelSerial := context.WithCancel(sigCtx)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			log.Println("voor data aan het wachten")
			time.Sleep(*interval)

			select {
			case <-serialCtx.Done():
				log.Println("klaar met lezen van serial")
				wg.Done()
				break
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
				log.Printf("%#v", tg)
			}
		}
	}()

	<-sigCtx.Done()
	sigStop()

	cancelSerial()
	wg.Wait()
	log.Println("stopping")
}
