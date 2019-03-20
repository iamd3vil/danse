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

// DNSCache is serialized and stored in the cache
type DNSCache struct {
	Msg       *dns.Msg
	CreatedAt time.Time
}

// dnsURLs holds urls for multiple doh providers and the last doh provider used
type dnsURLs struct {
	// Slice of URLs
	urls []string

	// Last used index
	lIndex int

	// Mutex
	lock *sync.Mutex
}

// GetDNSResponse contacts DOH provider and formats the reply into dns message
func GetDNSResponse(m *dns.Msg, httpClient *http.Client, durls *dnsURLs) (*dns.Msg, error) {
	b, err := m.Pack()
	if err != nil {
		return &dns.Msg{}, err
	}

	durls.lock.Lock()

	resp, err := httpClient.Post(durls.urls[durls.lIndex], "application/dns-message", bytes.NewBuffer(b))
	if err != nil {
		return &dns.Msg{}, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Response from DOH provider has status code: %d", resp.StatusCode)
		return &dns.Msg{}, errors.New("Error from DOH provider")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &dns.Msg{}, nil
	}

	r := &dns.Msg{}
	err = r.Unpack(body)

	// Increase last index
	durls.lIndex++

	if durls.lIndex == len(durls.urls) {
		durls.lIndex = 0
	}

	durls.lock.Unlock()

	return r, err
}
