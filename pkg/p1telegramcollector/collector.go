package p1telegramcollector

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"regexp"
	"sync"
	"time"
)

type Collector struct {
	bmu sync.RWMutex
	b   *bytes.Buffer

	r io.Reader
	w io.Writer
	s *bufio.Scanner

	stop chan bool
	C    chan *bytes.Buffer
}

var (
	telegramR = regexp.MustCompile("(?sm)/[^!]*[!][0-9A-Z]{4}")
)

func New() *Collector {
	return &Collector{
		stop: make(chan bool),
		b:    new(bytes.Buffer),
		C:    make(chan *bytes.Buffer),
	}
}

func (c *Collector) Write(b []byte) (int, error) {
	c.bmu.Lock()
	defer c.bmu.Unlock()
	return c.b.Write(b)
}

func (c *Collector) Run(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.C <- nil
				close(c.C)
				return
			default:
			}

			func() {
				c.bmu.RLock()
				defer c.bmu.RUnlock()
				defer time.Sleep(500 * time.Millisecond)

				b := c.b.Bytes()
				loc := telegramR.FindIndex(b)
				if loc != nil {
					// nieuwste telegram
					telegram := b[loc[0]:loc[1]]

					// ervolgend data
					c.b = new(bytes.Buffer)
					c.b.Write(b[loc[1]+1:])

					c.C <- bytes.NewBuffer(telegram)
				}
			}()
		}
	}()
}
