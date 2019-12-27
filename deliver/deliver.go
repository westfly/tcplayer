// Copyright © 2017 feilengcui008 <feilengcui008@gmail.com>.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Deliver stands for a remote host to send traffic to
package deliver

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

type ModeType int

const (
	ModeRequest ModeType = iota
	ModeRaw
)
const (
	TBinaryProtocol = iota
	TCompactProtocol
)

type DeliverConfig struct {
	IsLong       bool
	Concurrency  int
	RemoteAddr   string
	Last         int
	Clone        int
	ProtocolType int
	Mode         ModeType
}

type Deliver struct {
	Config  *DeliverConfig
	Stat    *Stat
	Clients []*Client
	Ctx     context.Context
	C       chan []byte
}

func (d *Deliver) startClient(ch chan struct{}) {
	for i := 0; i < d.Config.Concurrency; i++ {
		clientConfig := &ClientConfig{
			RemoteAddr: d.Config.RemoteAddr,
			Clone:      d.Config.Clone,
			IsLong:     d.Config.IsLong,
		}
		client, err := NewClient(d.Ctx, clientConfig)
		if err != nil {
			log.Errorf("create client %d failed: %v", i, err)
			continue
		}
		d.Clients = append(d.Clients, client)
	}
	ch <- struct{}{}
}

func (d *Deliver) deliverRequest() {
	d.Stat.StartTime = time.Now()
	d.Stat.LastStatTime = time.Now()
	for {
		select {
		case <-d.Ctx.Done():
			return
		case req := <-d.C:
			for i := 0; i < d.Config.Clone+1; i++ {
				d.Stat.TotalRequest++
				now := time.Now()
				if now.After(d.Stat.LastStatTime.Add(time.Second * 1)) {
					d.Stat.RequestPerSecond = d.Stat.TotalRequest - d.Stat.LastTotalRequest
					d.Stat.LastTotalRequest = d.Stat.TotalRequest
					d.Stat.LastStatTime = now
					log.Infof("deliver total reqs %d, %d reqs/s", d.Stat.TotalRequest, d.Stat.RequestPerSecond)
				}
				// choose a random client
				idx := rand.Int() % len(d.Clients)
				d.Clients[idx].S.Data() <- req
				log.Debugf("send packets to %s with connection %d", d.Config.RemoteAddr, idx)
			}
		}
	}
}

func (d *Deliver) Run() error {
	if d.Config == nil {
		err := fmt.Errorf("deliver config is not set")
		return err
	}
	// we start clients only with ModeRequest
	if d.Config.Mode == ModeRequest {
		ch := make(chan struct{})
		go d.startClient(ch)
		<-ch
		go d.deliverRequest()
	}
	select {
	case <-d.Ctx.Done():
		return fmt.Errorf("deliver stopped by context done")
	}
}
func (d *Deliver) GetRandomSender() (Sender, error) {
	len := len(d.Clients)
	if len == 0 {
		return nil, fmt.Errorf("empty client, pls check")
	}
	idx := rand.Int() % len
	return d.Clients[idx].S, nil
}
func NewDeliver(ctx context.Context, config *DeliverConfig) (*Deliver, error) {
	if len(config.RemoteAddr) == 0 {
		err := fmt.Errorf("deliver config not set RemoteAddrs")
		return nil, err
	}
	log.Debugf("deliver config %#v", config)
	d := &Deliver{
		Config:  config,
		C:       make(chan []byte),
		Clients: []*Client{},
		Stat:    &Stat{},
		Ctx:     ctx,
	}
	go d.Run()
	return d, nil
}
