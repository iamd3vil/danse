package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/miekg/dns"
)

const cloudflareDNSURL = "https://cloudflare-dns.com/dns-query"

type env struct {
	httpClient *http.Client
	url        string
	db         *badger.DB
}

func main() {
	port := flag.String("port", "53", "Port for DNS server")
	url := flag.String("url", cloudflareDNSURL, "URL for DoH resolver")
	addr := flag.String("addr", "127.0.0.1", "Address to bind")
	cachePath := flag.String("cache", "/tmp/danse/", "Path for storing cache for Danse")

	flag.Parse()

	opts := badger.DefaultOptions
	opts.Dir = *cachePath
	opts.ValueDir = *cachePath
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalln("Couldn't create cache in /tmp/danse")
	}

	defer db.Close()

	// Clear DB before starting the server
	db.DropAll()
	// Schedule a GC for badger at a periodic interval
	go runGC(db)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	dnsServer := &dns.Server{Addr: fmt.Sprintf("%s:%s", *addr, *port), Net: "udp"}

	e := env{httpClient: httpClient, url: *url, db: db}

	dns.HandleFunc(".", e.handleDNS)

	log.Fatalln(dnsServer.ListenAndServe())

}

func (e *env) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	log.Println("Got DNS query for ", r.Question[0].String())

	// Check in cache if it exists
	err := e.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(r.Question[0].String()))
		if err != nil && err != badger.ErrKeyNotFound {
			log.Printf("Error: %v", err)
			return err
		}

		if err == badger.ErrKeyNotFound {
			respMsg, err := GetDNSResponse(r, e.httpClient, e.url)
			if err != nil {
				log.Printf("Something wrong with resp: %v", err)
				return err
			}

			// Put in cache
			key := r.Question[0].String()

			// Get minimum TTL of all the records
			ttl := minTTL(respMsg)

			respPacked, err := respMsg.Pack()
			if err != nil {
				return err
			}

			dnsCache := DNSCache{Msg: respPacked, CreatedAt: time.Now()}

			var packed bytes.Buffer

			// Serializing the response to store in Badger
			err = gob.NewEncoder(&packed).Encode(&dnsCache)
			if err != nil {
				return err
			}

			err = txn.SetWithTTL([]byte(key), packed.Bytes(), ttl)
			if err != nil {
				log.Printf("Couldn't save to cache because: %v", err)
			}

			w.WriteMsg(respMsg)
			return nil
		}

		resp, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		// DNSCache stored in badger
		cResp := &DNSCache{}
		err = gob.NewDecoder(bytes.NewBuffer(resp)).Decode(cResp)
		if err != nil {
			return err
		}
		respMsg := &dns.Msg{}
		err = respMsg.Unpack(cResp.Msg)
		if err != nil {
			return err
		}
		// Set ID of the response to request ID
		respMsg.MsgHdr.Id = r.MsgHdr.Id

		// Adjust TTL according to the current time
		w.WriteMsg(adjustTTL(respMsg, cResp.CreatedAt))
		return nil
	})
	if err != nil {
		log.Printf("Error in handle DNS: %v", err)
	}
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

func runGC(db *badger.DB) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		db.RunValueLogGC(0.7)
		log.Printf("Ran badger GC")
	}
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
