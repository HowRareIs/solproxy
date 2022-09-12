# HowRare Solana RPC proxy
HowRare Solana RPC proxy is there to allow project creators to freely route solana RPC calls to different solana nodes utilizing prioritization and capping.

It allows to:
- Route requests between fast local node(s) with partial chain data and remote node(s) with full chain history
- Makes it possible to keep requests below allocated limits with per-function, per-request and per-transfer capping
- Spread the load across many nodes/providers
- Automatically detects and skips failed or overloaded/timeouting nodes
- Automatically re-do failed requests on different node if they timed out or returned an error

## Building the software
Run following commands to build for windows / linux. Golang 1.18 required. 
<pre>cd solproxy/gosol/main
go build main.go
</pre>

## Node types
There are 2 node types defined
- Public - this node stores full archival chain data
- Private - fast local node with partial chain data
If you don't need to distinct and you want to use the proxy just to route your requests to different providers for loadbalancing / failover - you can setup all nodes as a private type.

## Throttling
There is automatic throttling/routing implemented. If node is throttled the request will be routed to different node. If all available nodes are throttled so there's no node to pick to run the request - you will get response with error attribute and issue description.
```json
{"error":"Throttled public node, please wait","throttle_info":{"requests":{"description":"requests made","max":99,"value":3},"requests_fn":{"description":"requests made calling single function","max":39,"value":3},"received":{"description":"bytes received","max":1000000,"value":4735645}},"throttle_timespan_seconds":12,"throttled":true,"throttled_comment":"Too much data received 4735645/1000000"}
```

## Simple mode
There are 2 modes of operation. In **simple mode** you connect to the proxy like you'd do to a normal solana node, and your requests will get routed between available **private** nodes. You'll add all nodes as private.

## Configuration
```json
{
"BIND_TO": "h127.0.0.1:7778,h8.8.8.8:7778,",

"FORCE_START":true,
"DEBUG":false,
"VERBOSE":false,
"RUN_SERVICES":"*",

"SOL_NODES":[{"url":"http://127.0.0.1:8899", "public":false, "score_modifier":-90000},
		{"url":"https://api.mainnet-beta.solana.com", "public":false, "throttle":"r,80,10;f,30,10;d,80000000,30", "probe_time":20},
		{"url":"https://solana-api.projectserum.com", "public":false, "throttle":"r,80,10;f,30,10;d,80000000,30", "probe_time":20}],
}
```
Configuration should be self-explanatory. You need to add h prefix before each IP the proxy will bind to. It'll listen for new connection on this IP/Port. There's a possibility to communicate with proxy using pure TCP by skipping the prefix.

Throttle can be configured in following way:
- r[equests],time_in_seconds,limit
- f[unction call],time_in_seconds,limit
- d[ata received],time_in_seconds,limit in bytes



## Accessing proxy information
http://127.0.0.1:7778/?action=server-status
You can access server-status page by using server-status action. There's also PHP script available to password-protect the status page so it can be accessible from outside.

http://127.0.0.1:7778/?action=getSolanaInfo
This url will return throttling status for public and private nodes.

http://127.0.0.1:7778/?action=getFirstAvailableBlock
Gets first available block.

### Advanced usage
In advanced mode you will access the proxy using HTTP. Requests will get routed to public or private node depends on if they need archival data to be fullfilled. Private node gets picked by default, then if it has no data needed to fullfill the request - it'll be re-done on public node. 

You can also add &public=1 or &private=1 to force public or private node to be picked to run the request. The preferred way of interacting with proxy when using HTTP is using &public=1 when you need to run a request which will require archival data and skip adding &private as private is the default anyway and by adding it you'll just disable fallback to a public node for given request.

http://127.0.0.1:7778/?action=getBlock&block=95535092

http://127.0.0.1:7778/?action=getTransaction&hash=4P4Gpz2BEqFQ2p4MqWKqPM8ZD6FFbJsM9BUrvAssrTybUrFxZxRfESE4CUbNBsMx655QEXhup8UMACKZ37wrSfGH

http://127.0.0.1:7778/?action=getBalance&pubkey=2ExPNqnptwVQ1h1LNkeF1o1CahHMX1AjsNxi7FJXXWbT

### Advanded mode, raw calls
There is a possibility to run any solana RPC call using action=solanaRaw. One private node gets picked first, then the request can be routed to public node if the private node has no required chain data and returns null. However that's not quaranteed, as some requrest will not return null when data is (partially) missing.

http://127.0.0.1:7778/?action=solanaRaw&method=getConfirmedBlock&params=[94135095]

