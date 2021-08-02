package main

import "github.com/miekg/dns"

type DNSClient interface {
	GetDNSResponse(m *dns.Msg) (*dns.Msg, error)
}
