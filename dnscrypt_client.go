package main

import (
	"log"
	"sync"
	"time"

	"github.com/ameshkov/dnscrypt/v2"
	"github.com/miekg/dns"
)

type DCryptClient struct {
	client        *dnscrypt.Client
	resolverInfos []*dnscrypt.ResolverInfo

	// Last used index
	lIndex int

	logQueries bool

	sync.Mutex
}

func (c *DCryptClient) GetDNSResponse(msg *dns.Msg) (*dns.Msg, error) {
	if len(msg.Question) == 0 {
		return nil, nil
	}

	c.Lock()

	rinfo := c.resolverInfos[c.lIndex]

	// Increase last index
	c.lIndex++

	if c.lIndex == len(c.resolverInfos) {
		c.lIndex = 0
	}

	c.Unlock()

	if c.logQueries {
		log.Printf("Sending to %s for query: %s", rinfo.ProviderName, msg.Question[0].String())
	}
	return c.client.Exchange(msg, rinfo)
}

func NewDNSCryptClient(stamps []string, logQueries bool) (*DCryptClient, error) {
	c := dnscrypt.Client{
		Net:     "udp",
		Timeout: 10 * time.Second,
	}
	client := DCryptClient{
		client:        &c,
		resolverInfos: make([]*dnscrypt.ResolverInfo, 0, len(stamps)),
		logQueries:    logQueries,
	}

	for _, stamp := range stamps {
		resolverInfo, err := c.Dial(stamp)
		if err != nil {
			return nil, err
		}

		client.resolverInfos = append(client.resolverInfos, resolverInfo)
	}

	return &client, nil
}
