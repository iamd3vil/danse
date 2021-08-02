package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/miekg/dns"
)

type DohClient struct {
	httpClient *http.Client

	// Slice of URLs
	urls []string

	// Last used index
	lIndex int

	logQueries bool

	sync.Mutex
}

func (c *DohClient) GetDNSResponse(msg *dns.Msg) (*dns.Msg, error) {
	b, err := msg.Pack()
	if err != nil {
		return &dns.Msg{}, err
	}

	c.Lock()

	url := c.urls[c.lIndex]

	// Increase last index
	c.lIndex++

	if c.lIndex == len(c.urls) {
		c.lIndex = 0
	}

	c.Unlock()

	if c.logQueries {
		log.Printf("Sending to %s for query: %s", url, msg.Question[0].String())
	}

	resp, err := c.httpClient.Post(url, "application/dns-message", bytes.NewBuffer(b))
	if err != nil {
		return &dns.Msg{}, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Response from DOH provider has status code: %d", resp.StatusCode)
		return &dns.Msg{}, errors.New("error from DOH provider")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &dns.Msg{}, nil
	}

	r := &dns.Msg{}
	err = r.Unpack(body)

	return r, err
}

func NewDOHClient(c *http.Client, urls []string, logQueries bool) (*DohClient, error) {
	return &DohClient{
		httpClient: c,
		urls:       urls,
		logQueries: logQueries,
	}, nil
}
