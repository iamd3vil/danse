# Danse

Danse is a DNS resolver which receives packets over conventional DNS(UDP) and resolves it by talking to another resolver over DNS over HTTPS(DoH). DoH would reduce any snooping by ISP or any middlemen since the requests would be encrypted.

This would allow any application which doesn't support DoH still use DoH. Danse is supposed to be run locally or on a local network. There is no point running this over internet since DNS queries then wouldn't be encrypted between your device and Danse.

## Usage

```shell
$ danse -config /etc/danse/config.toml
```

Sample config:

```toml
# Address for danse to listen on.
bind_address = "127.0.0.1:5454"

# Should the answers be cached according to ttl. Default is true.
cache = true

# Log level
log_level = "info"

# Logs all queries to stdout. Default is false.
log_queries = true

# Only used for resolving resolver url. No queries received by danse will be sent here. Default is 9.9.9.9:53
bootstrap_address = "1.1.1.1:53"

# Urls for resolvers.
[resolver]
urls = ["https://dns.quad9.net/dns-query", "https://cloudflare-dns.com/dns-query"]
```

A sample config file with all the fields can be found at `config.sample.toml`.

## TODO

- [X] Caching
- [X] Load Balance to multiple DoH providers for improved privacy
- [X] Option to log queries
- [X] Option to provide a bootstrap DNS server for resolving the given urls