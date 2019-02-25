package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

// DNSCache is serialized and stored in the cache
type DNSCache struct {
	Msg       []byte
	CreatedAt time.Time
}

// GetDNSResponse contacts DOH provider and formats the reply into dns message
func GetDNSResponse(m *dns.Msg, httpClient *http.Client, dnsURL string) (*dns.Msg, error) {
	b, err := m.Pack()
	if err != nil {
		return &dns.Msg{}, err
	}

	resp, err := httpClient.Post(dnsURL, "application/dns-message", bytes.NewBuffer(b))
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

	return r, err
}
