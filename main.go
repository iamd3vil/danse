package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

const cloudflareDNSURL = "https://cloudflare-dns.com/dns-query"

type env struct {
	httpClient *http.Client
	url        string
}

func main() {
	port := flag.String("port", "53", "Port for DNS server")
	url := flag.String("url", cloudflareDNSURL, "URL for DoH resolver")

	flag.Parse()

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	dnsServer := &dns.Server{Addr: fmt.Sprintf(":%s", *port), Net: "udp"}

	e := env{httpClient: httpClient, url: *url}

	dns.HandleFunc(".", e.handleDNS)

	log.Fatalln(dnsServer.ListenAndServe())

}

func (e *env) handleDNS(w dns.ResponseWriter, r *dns.Msg) {
	respMsg, err := GetDNSResponse(r, e.httpClient, e.url)
	if err != nil {
		log.Printf("Something wrong with resp: %v", err)
		return
	}
	w.WriteMsg(respMsg)
}
