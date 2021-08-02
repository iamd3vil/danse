package main

import (
	"time"

	"github.com/miekg/dns"
)

// DNSInCache is serialized and stored in the cache
type DNSInCache struct {
	Msg       *dns.Msg
	CreatedAt time.Time
}
