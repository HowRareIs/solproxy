# Solproxy advanced usage

Solproxy defines some additional APIs, which will allow for better control. You will access the proxy using HTTP. Requests will get routed to public or private node depends on if they need archival data to be fullfilled. Private node gets picked by default, then if it has no data needed to fullfill the request - it'll be re-done on public node. 

You can also add &public=1 or &private=1 to force public or private node to be picked to run the request. The preferred way of interacting with proxy when using HTTP is using &public=1 when you need to run a request which will require archival data and skip adding &private as private is the default anyway and by adding it you'll just disable fallback to a public node for given request.

http://127.0.0.1:7778/?action=getBlock&block=95535092

http://127.0.0.1:7778/?action=getTransaction&hash=4P4Gpz2BEqFQ2p4MqWKqPM8ZD6FFbJsM9BUrvAssrTybUrFxZxRfESE4CUbNBsMx655QEXhup8UMACKZ37wrSfGH

http://127.0.0.1:7778/?action=getBalance&pubkey=2ExPNqnptwVQ1h1LNkeF1o1CahHMX1AjsNxi7FJXXWbT

### Advanded mode, raw calls
There is a possibility to run any solana RPC call using action=solanaRaw. One private node gets picked first, then the request can be routed to public node if the private node has no required chain data and returns null. However that's not quaranteed, as some requrest will not return null when data is (partially) missing.

http://127.0.0.1:7778/?action=solanaRaw&method=getConfirmedBlock&params=[94135095]
