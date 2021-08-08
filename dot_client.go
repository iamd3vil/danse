package main

import (
	"log"
	"net"
	"sync"

	"github.com/miekg/dns"
)

type DOTClient struct {
	urls   []string
	conns  map[string]*dns.Conn
	client *dns.Client

	// Last used index
	lIndex int

	logQueries bool
	sync.Mutex
}

func (c *DOTClient) GetDNSResponse(msg *dns.Msg) (*dns.Msg, error) {
	c.Lock()

	url := c.urls[c.lIndex]

	// Increase last index
	c.lIndex++

	if c.lIndex == len(c.urls) {
		c.lIndex = 0
	}

	c.Unlock()

	log.Printf("log queries: %v", c.logQueries)

	if c.logQueries {
		log.Printf("Sending to %s for query: %s", url, msg.Question[0].String())
	}

	var r *dns.Msg

	var makeconn bool
	for i := 0; i < 5; i++ {
		conn, err := c.getConn(url, makeconn)
		if err != nil {
			return &dns.Msg{}, err
		}

		r, _, err = c.client.ExchangeWithConn(msg, conn)
		if err != nil {
			makeconn = true
			continue
		}
		break
	}
	return r, nil
}

func (c *DOTClient) getConn(url string, makeconn bool) (*dns.Conn, error) {
	c.Lock()
	defer c.Unlock()

	if conn, ok := c.conns[url]; ok {
		if makeconn {
			goto makeConn
		}
		return conn, nil
	}

makeConn:
	conn, err := c.client.Dial(url)
	if err != nil {
		return nil, err
	}

	c.conns[url] = conn
	return conn, nil
}

func NewDOTClient(urls []string, dialer *net.Dialer, logQueries bool) (*DOTClient, error) {
	return &DOTClient{
		urls: urls,
		client: &dns.Client{
			Net:    "tcp-tls",
			Dialer: dialer,
		},
		conns:      map[string]*dns.Conn{},
		logQueries: logQueries,
	}, nil
}
