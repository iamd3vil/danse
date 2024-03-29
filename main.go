package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/miekg/dns"
)

var buildString string

type env struct {
	cache  *lru.Cache
	cfg    Config
	client DNSClient
}

func main() {
	cfgPath := flag.String("config", "config.toml", "Path to config file")
	version := flag.Bool("version", false, "Version")
	flag.Parse()

	if *version {
		fmt.Println(buildString)
		os.Exit(0)
	}

	cfg, err := initConfig(*cfgPath)
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	maxCacheItems := 512
	if cfg.Cache.MaxItems != 0 {
		maxCacheItems = cfg.Cache.MaxItems
	}

	e := env{
		cfg: cfg,
	}

	// Initialize cache
	if cfg.Cache.Cache {
		cache, err := lru.New(maxCacheItems)
		if err != nil {
			log.Fatalln("Couldn't create cache: ", err)
		}

		e.cache = cache
	}

	// Make a dialer which resolves url with bootstrap address.
	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: 5 * time.Second,
				}
				return d.DialContext(ctx, "udp", cfg.BootstrapAddress)
			},
		},
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}

	var (
		client DNSClient
	)

	switch cfg.Resolver.Type {
	case "doh":
		client, err = getDOHDnsClient(dialContext, cfg)
		if err != nil {
			log.Fatalf("Couldn't create DNS client: %v", err)
		}

		e.client = client
	case "dot":
		client, err = NewDOTClient(cfg.Resolver.Urls, dialer, cfg.Log.LogQueries)
		if err != nil {
			log.Fatalf("Couldn't create DNS client: %v", err)
		}

		e.client = client
	case "dnscrypt":
		client, err := NewDNSCryptClient(cfg.Resolver.Urls, cfg.Log.LogQueries)
		if err != nil {
			log.Fatalf("Couldn't create DNSCrypt client: %v", err)
		}

		e.client = client
	}

	// Start the DNS server.
	dnsServer := &dns.Server{Addr: cfg.BindAddress, Net: "udp"}

	dns.HandleFunc(".", e.handleDNS)

	log.Fatalln(dnsServer.ListenAndServe())
}

func (e *env) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) == 0 {
		return
	}

	if e.cfg.Log.LogQueries {
		log.Println("Got DNS query for ", r.Question[0].String())
	}

	cacheKey := r.Question[0].String()
	// Check if we should check cache or not
	if !e.cfg.Cache.Cache {
		e.getAndSendResponse(w, r, cacheKey)
		return
	}

	// Check if the key is already in cache
	val, ok := e.cache.Get(cacheKey)
	if !ok {
		e.getAndSendResponse(w, r, cacheKey)
		return
	}

	cResp := val.(DNSInCache)

	// Check if this record is expired
	ttl := minTTL(cResp.Msg)

	if time.Since(cResp.CreatedAt) > ttl {
		e.getAndSendResponse(w, r, cacheKey)
		return
	}

	resp := cResp.Msg

	resp.MsgHdr.Id = r.MsgHdr.Id

	w.WriteMsg(adjustTTL(resp, cResp.CreatedAt))
}

func (e *env) getAndSendResponse(w dns.ResponseWriter, r *dns.Msg, cacheKey string) {
	respMsg, err := e.client.GetDNSResponse(r)
	if err != nil {
		log.Printf("Something wrong with resp: %v", err)
		return
	}

	// Put it in cache
	if e.cfg.Cache.Cache {
		dnsCache := DNSInCache{Msg: respMsg, CreatedAt: time.Now()}
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
		ttl := math.Round(time.Until(expiresAt).Seconds())
		adj.Answer[i].Header().Ttl = uint32(ttl)
	}
	return &adj
}

func getDOHDnsClient(dialContext func(ctx context.Context, network string, addr string) (net.Conn, error), cfg Config) (DNSClient, error) {
	transport := http.DefaultTransport.(*http.Transport)
	transport.DialContext = dialContext

	httpClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	return NewDOHClient(httpClient, cfg.Resolver.Urls, cfg.Log.LogQueries)
}
