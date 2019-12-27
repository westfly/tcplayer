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

// Client stands for a client which send requests to remote
// server, client can clone at request level, since request
// is not related to the underlining tcp packet sequence, so
// more flexible.
package deliver

import (
	"context"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
)

type ClientConfig struct {
	RemoteAddr string
	IsLong     bool
	Clone      int
}

type Client struct {
	Idx    int
	Config *ClientConfig
	S      Sender
	Ctx    context.Context
	Stat   *Stat
}

func NewClient(ctx context.Context, c *ClientConfig) (*Client, error) {
	var (
		client  = &Client{Config: c}
		creator = NewLongConnSender
	)
	if !c.IsLong {
		creator = NewShortConnSender
	}
	host, _, _ := net.SplitHostPort(c.RemoteAddr)
	remote_model := net.ParseIP(host).To4()
	if remote_model == nil {
		log.Errorf("parse remote %s failed will use localfile_model", c.RemoteAddr)
		creator = NewLocalFileWriter
	}
	s, err := creator(ctx, 1, c.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("create client failed: %s", err)
	}
	client.S = s
	return client, nil
}
