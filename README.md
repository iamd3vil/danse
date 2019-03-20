# Danse

Danse is a DNS resolver which receives packets over conventional DNS(UDP) and resolves it by talking to another resolver over DNS over HTTPS(DoH). DoH would reduce any snooping by ISP or any middlemen since the requests would be encrypted.

This would allow any application which doesn't support DoH still use DoH. Danse is supposed to be run locally or on a local network. There is no point running this over internet since DNS queries then wouldn't be encrypted between your device and Danse.

## Usage

```
Usage of ./danse:
  -addr string
        Address to bind (default "127.0.0.1")
  -cache
        DNS response caching
  -port string
        Port for DNS server (default "53")
  -url string
        URLs for DoH resolvers seperated by comma (default "https://cloudflare-dns.com/dns-query")
```

## TODO

- [X] Caching
- [X] Load Balance to multiple DoH providers for improved privacy
