package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// DNSInCache is serialized and stored in the cache
type DNSInCache struct {
	Msg       *dns.Msg
	CreatedAt time.Time
}

// dnsURLs holds urls for multiple doh providers and the last doh provider used
type dnsURLs struct {
	// Slice of URLs
	urls []string

	// Last used index
	lIndex int

	sync.Mutex
}

// GetDNSResponse contacts DOH provider and formats the reply into dns message
func (e *env) GetDNSResponse(m *dns.Msg, httpClient *http.Client, durls *dnsURLs) (*dns.Msg, error) {
	b, err := m.Pack()
	if err != nil {
		return &dns.Msg{}, err
	}

	durls.Lock()

	url := durls.urls[durls.lIndex]

	// Increase last index
	durls.lIndex++

	if durls.lIndex == len(durls.urls) {
		durls.lIndex = 0
	}

	durls.Unlock()

	if e.cfg.LogQueries {
		log.Printf("Sending to %s for query: %s", url, m.Question[0].String())
	}

	resp, err := httpClient.Post(url, "application/dns-message", bytes.NewBuffer(b))
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
