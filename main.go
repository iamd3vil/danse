package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/miekg/dns"
)

const cloudflareDNSURL = "https://cloudflare-dns.com/dns-query"

type tlsClients struct {
	sync.Mutex
	clients map[uint16]chan *dns.Msg
}

type env struct {
	httpClient *http.Client
	dnsUrls    *dnsURLs
	cache      *lru.Cache
	shdCache   bool // Should we cache DNS responses or not
	dot        bool // If DOT is enabled

	tlsConn    *tls.Conn
	connNotify chan int
	query      chan *dns.Msg

	// Contains map of dns queries and reply channels
	tlsClients tlsClients
}

var dot bool

func main() {
	port := flag.String("port", "53", "Port for DNS server")
	urls := flag.String("url", cloudflareDNSURL, "URLs for DoH resolvers seperated by comma")
	addr := flag.String("addr", "127.0.0.1", "Address to bind")
	shdCache := flag.Bool("cache", false, "DNS response caching")
	dotls := flag.Bool("dot", false, "DNS Over TLS instead of DOH")
	dotaddr := flag.String("dotaddr", "1.1.1.1", "DNS Over Tls Address(IP Address)")
	dotname := flag.String("dotname", "cloudflare-dns.com", "DOT Server Name")

	flag.Parse()

	// Initialize cache
	cache, err := lru.New(512)
	if err != nil {
		log.Fatalln("Couldn't create cache: ", err)
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	dnsServer := &dns.Server{Addr: fmt.Sprintf("%s:%s", *addr, *port), Net: "udp"}

	dnsurls := &dnsURLs{
		urls: strings.Split(*urls, ","),
	}

	e := env{
		httpClient: httpClient,
		dnsUrls:    dnsurls,
		cache:      cache,
		shdCache:   *shdCache,
		dot:        *dotls,
	}

	// If DOT is enabled instead of DOH, start go routines
	if *dotls {
		e.query = make(chan *dns.Msg, 1000)
		e.tlsClients = tlsClients{
			clients: make(map[uint16]chan *dns.Msg),
		}
		go e.makeTLSConnection(fmt.Sprintf("%s:853", *dotaddr), *dotname)
	}

	dns.HandleFunc(".", e.handleDNS)

	log.Fatalln(dnsServer.ListenAndServe())
}

func (e *env) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	log.Println("Got DNS query for ", r.Question[0].String())

	cacheKey := r.Question[0].String()
	// Check if we should check cache or not
	if !e.shdCache {
		e.getAndSendResponse(w, r, cacheKey)
		return
	}

	// Check if the key is already in cache
	val, ok := e.cache.Get(cacheKey)
	if !ok {
		e.getAndSendResponse(w, r, cacheKey)
		return
	}

	cResp := val.(DNSCache)

	// Check if this record is expired
	ttl := minTTL(cResp.Msg)

	if time.Now().Sub(cResp.CreatedAt) > ttl {
		e.getAndSendResponse(w, r, cacheKey)
		return
	}

	resp := cResp.Msg

	resp.MsgHdr.Id = r.MsgHdr.Id

	w.WriteMsg(adjustTTL(resp, cResp.CreatedAt))
	return
}

func (e *env) getAndSendResponse(w dns.ResponseWriter, r *dns.Msg, cacheKey string) {
	respMsg, err := e.GetDNSResponse(r, e.httpClient, e.dnsUrls)
	if err != nil {
		log.Printf("Something wrong with resp: %v", err)
		return
	}

	// Put it in cache
	if e.shdCache {
		dnsCache := DNSCache{Msg: respMsg, CreatedAt: time.Now()}
		e.cache.Add(cacheKey, dnsCache)
	}

	w.WriteMsg(respMsg)
}

func minTTL(m *dns.Msg) time.Duration {
	if len(m.Answer) >= 1 {
		ttl := m.Answer[0].Header().Ttl
		for _, a := range m.Answer {
			if a.Header().Ttl < ttl {
				ttl = a.Header().Ttl
			}
		}
		return time.Duration(ttl) * time.Second
	}
	return 0
}

func adjustTTL(m *dns.Msg, createdAt time.Time) *dns.Msg {
	adj := *m
	for i := 0; i < len(m.Answer); i++ {
		expiresAt := createdAt.Add(time.Duration(m.Answer[i].Header().Ttl) * time.Second)
		ttl := math.Round(expiresAt.Sub(time.Now()).Seconds())
		adj.Answer[i].Header().Ttl = uint32(ttl)
	}
	return &adj
}
