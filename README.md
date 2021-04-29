# Danse

Danse is a DNS resolver which receives packets over conventional DNS(UDP) and resolves it by talking to another resolver over DNS over HTTPS(DoH). DoH would reduce any snooping by ISP or any middlemen since the requests would be encrypted.

This would allow any application which doesn't support DoH still use DoH. Danse is supposed to be run locally or on a local network. There is no point running this over internet since DNS queries then wouldn't be encrypted between your device and Danse.

## Usage

A `config.toml` needs to be present from the path the `danse` binary is running. 

Sample config:

```toml
bind_address = "127.0.0.1:5454"
cache = true
log_level = "info"
log_queries = true

[resolver]
urls = ["https://dns.quad9.net/dns-query", "https://cloudflare-dns.com/dns-query"]
```

## TODO

- [X] Caching
- [X] Load Balance to multiple DoH providers for improved privacy
- [X] Option to log queries
- [ ] Option to provide a bootstrap DNS server for resolving the given urls