# Danse

Danse is a DNS resolver which receives packets over conventional DNS(UDP) and resolves it by talking to another resolver over DNS over HTTPS(DoH).

This would allow any application which doesn't support DoH still use DoH. Danse is supposed to be run locally or on a local network. There is no point running this over internet since DNS queries then wouldn't be encrypted between your device and Danse.