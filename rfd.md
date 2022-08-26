---
author: Daniele Pagano (daniele@danielepagano.com)
state: draft
---

# RFD - Basic TCP Load Balancer (Teleport Challenge)

## What & Why

Author is to implement a basic TCP Load Balancer (Level 4) by request of the Teleport Systems Team, 
as described [here](https://github.com/gravitational/careers/blob/main/challenges/systems/challenge-2.md#level-4).

The system will be delivered as a reusable library and server (plus relevant tests) written in Go and fully contained within this repository.

## Details

### Library: lbproxy

The core of the application is the reusable Go package `lbproxy`, which implements a configurable least-connections TCP request forwarder with a rate-limiting.

At the highest level, when using lbproxy, the clients will create an instance of the `Application` type for each application 
(group of upstream server addresses) that they would like to load-balance.
When creating an `Application` instance, clients will provide an `ApplicationConfig`:

```go
package lbproxy

// ApplicationConfig initializes an Application instance
type ApplicationConfig struct {
	name      string            // App name for diagnostic purposes
	upstreams []UpstreamServer  // Upstream servers to load-balance
}

// UpstreamServer describes a server being load-balanced
// Note: using a struct allows non-breaking extensions to add other properties, like certificates or relative 
// upstream weights for a more advanced load-balancing algorithm.
type UpstreamServer struct {
	address string
}
```

To support rate-limiting, clients can create a `RateLimitManager` instance, which can track various statistics, and signal when a rate has been exceeded. 
This is accomplished by sending the instance an event (connection opened, data transferred, etc.) 
and then seeing if the response indicates the rate limit has been exceeded. 
It is crucial that the `RateLimitManager` events are safe to dispatch across threads, as multiple connection in parallel will
update rate limit statistics and check if the rate has been exceeded.

- The `RateLimitManager` can provide client-level, app-level, global, or other combinations of rate-limiting, depending on how many instances the server creates, and how they are passed to `Application` when a connection is opened.
- For this project, we'll create a simple implementation that tracks the number of connections open over a time period, 
ensuring they do not exceed a max concurrent open cap (important to avoid slow-rate attacks), 
and that no more than a certain amount are opened over a time period.
- Possible extensions include enforcing bandwidth usage, connection duration, and throttling VS simple connection rejection.
We could also nest instances, which would allows us to stack multiple policies transparently from the proxy code itself. 

Once the `Application` instance is created, it will be ready to accept client connections. 
To do so, the client will call `submitConnection` of the Application with:

- Incoming client connection, as reference to a `net.TCPConn` instance.
- An instance of `RateLimitManager`, which provides rate-limiting for scope as controlled by caller.

The `Application` will maintain a concurrent map from the upstream address to the number of active connections for the address.
When a connection is requested, the proxy library will inform `RateLimitManager` of the event, and if the response allows a connection, 
it will iterate through the upstreams looking for the one lowest number of open connections 
(keeping the upstreams ordered, like with a treemap, would be overkill for this implementation), increase the connection count, then attempt to dial the upstream. 
If dial fails (or when the connection is closed), the number of open connections for the upstream will be lowered 
(this optimistic increment behavior lowers the number of operations against the map, since connection failures are considered the exception).

- The proxy uses simple buffering to transfer data to and from the upstreams; in a more mature system, this would be improved by
creating a faster and more controllable buffer (like a growable ring buffer); this is important as we want to limit memory
usage and reduce GC pauses, especially if the client, proxy, and upstream servers bandwidths are very different.

The Application itself will manage connection errors and take care of cleanly closing the connections to server and client.

- A more robust proxy would also support tracking health on each upstream: if a server does not respond to a request, 
it would be marked unhealthy and quarantined (removed from load-balancing list).
Servers could be re-added to the list after a certain time, although ideally we would implement a health-check we perform periodically, and add them if they pass.
Operational alerts would also be present in a production server for these events.

### Server

The server component of the system uses `lbproxy` to expose multiple applications, whilst providing strong security in a zero-trust network environment.
The server has the following primary responsibilities:

 1. Accepting incoming client connections
 2. Enforcing Security
 3. Providing configuration data to `lbproxy`

The server can be configured to proxy an arbitrary number of applications, each with an arbitrary number of upstream servers.
Each application will have a name (used to manage permissions), a port that the server will open to accept client connections, 
and a list of upstream servers, as defined in the `lbproxy` API.

The server will also be configured with a list of permitted client ids (resolved via our Authentication system described below) 
and which Application names they can access.

After reading the configuration, the server attempts to open each configured application port; assuming this succeeds, the proxy is online.
When a connection is accepted, the server will perform authentication and authorization; if they succeed, the application
will be proxied for the client. Within the scope of each application, the server will maintain an instance of `RateLimitManager` for each client, 
thus providing rate-limiting for that client plus application.

For this implementation, all configuration will be static, although it will be stored dedicated structures and injected on start,
so that they can be later be loaded from any combination of remote storage, configuration files, environment variables, and command line parameters.
As such, the server does not provide a UX: to run it, modify the static configuration data, build, start `cmd/main.go`, and connect to the open ports.

#### Production Limitations

This simple server is not highly available, has limited capacity, and does not preserve rate limits upon restart. Production-level improvements would include:

- Configuration and certificates should be held separately from the servers, at the very least using a system like Kubernetes secrets.
- The state of `RateLimitManager` can be distributed using a reliable distributed cache. Since we most likely do not need _exact_ rate-limiting, a cheaper and faster store that uses eventual consistency would be adequate.
- We can extend the same pattern as `RateLimitManager` to the system that counts open connection per upstream, thus allowing applications to be proxied by multiple current servers.
  - This is a bit more sensitive as we should not have "orphan" connections if a node fails, so each connection would need to have an id and reasonable TTL; if a node fails, the TTL would release the connection, otherwise the node would renew the TTL to hold the connection open.  
- Once the above is in place, we can use an address-redirecting load-balancer (not a proxy load-balancer, as we would just be moving traffic up one level) to act as a gateway to our cluster of proxies. 
  - A system like Zookeeper can be used to keep a real-time tally of live servers, although of course a higher-level management backplane like Kubernetes would be more practical.
- It may also be sensible to shard our system using client hashes to reduce blast radius; it may also simplify distributed storage, depending on its implementation.

### Security

The system will provide a minimalist but robust set of security features, based on [mTLS 1.3](https://www.rfc-editor.org/rfc/rfc8446.html).

#### Transport security

The connection between server and client will be secured by mTLS. 
For this application, we assume there is a single CA issuing client certificates, with a manual or automated process to issue them, 
e.g. as part of system provisioning or licensing process. In this implementation, for simplicity, 
we will generate and self-sign the certificates and load them from the local file system.

- Additionally, for each upstream, we could provide a client certificate to encrypt the traffic between the proxy and the upstream
(which is the server in this case), and also to ensure only authorized proxies can connect directly tp upstream servers. 
This may be omitted if this layer of security doesn't make sense for the application or environment in question.

#### Authentication

When a client connects with mTLS, we will inspect the name in the X509 certificate. Since we are assuming a since CA for the system, 
we can use the serial number as a unique client identifier. Because we also want to support authorization, 
this identifier is then cross-referenced to a user list, which is statically loaded for this implementation, 
but would be in a secure store linked to our CA/client-cert issuing system in a production implementation. 
If the serial number matches a known id, the client is authenticated.

#### Authorization

After validating the user id during authentication, we will load a set of claims for this user from the authorization store, 
which is just child data of the previous store used for authentication in this implementation. 
For this simple application, we will simply have a list of the allowed applications identifiers for this client.

- A simple extension to this system can include configurable rate limits for each application the client has access to, 
which can be tailored to usage, application type (e.g. sockets VS HTTP connections), or licensing agreements.
- Alternatively we could have bundled claims in the X509 certificate, but that makes system maintainability and certificate management more complex.
