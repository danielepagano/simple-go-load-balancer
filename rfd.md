---
author: Daniele Pagano (daniele@danielepagano.com)
state: draft
---

# RFD - Basic TCP Load Balancer (Teleport Challenge)

## What & Why

Author is to implement a basic TCP Load Balancer (Level 4) by request of the Teleport Systems Team, as described [here](https://github.com/gravitational/careers/blob/main/challenges/systems/challenge-2.md#level-4).

The system will be delivered as a reusable library and server (plus relevant tests) written in Go and fully contained within this repository.

## Details

### Library: lbproxy

The core of the application is the reusable Go library `lbproxy`, which implements a configurable least-connections TCP request forwarder with a per-client rate-limiter.

At the highest level, when using lbproxy, the clients will create an instance of the `LBProxyApp` type for each application (group of upstream server addresses) that they would like to load-balance.

When creating an `LBProxyApp` instance, clients will provide the following:

- An identifier for the application (`string`), for diagnostic purposes.
- A slice of upstream servers, each a `struct` containing at least the address `string` (IP or or hostname plus port) to dial. The struct allows non-breaking extensions to add other properties, like certificates or relative upstream weights for a more advanced load-balancing algorithm.

To support rate-limiting, clients can create a `RateLimitManager interface` instance. A `RateLimitManager` instance can track arbitrary statistics, and signal when a rate has been exceeded. This is done by sending the instance an event (connection opened, data transferred, etc.) and then seeing if the response indicates the rate limit has been exceeded.

- The `RateLimitManager` can provide client-level, app-level, global, or other combinations of rate-limiting, depending how many instances the server creates, and how they are passed to `LBProxyApp` when a connection is opened.
- For this project, we'll create a simple implementation that tracks the number of connections open over a time period, and keep an instance for each client, thus providing client-wide rate-limiting (client+app rate limiting would be achieved by keeping an instance for each app the client connects to).
- Possible extensions include enforcing bandwidth usage, throttling VS simple connection rejection, and custom configuration for these features on every instance.

Once the `LBProxyApp` instance is created, it will be ready to accept client connections. Each connection will be a struct with the following details:

- Incoming client connection, as reference to a `net.TCPConn` instance.
- An instance of `RateLimitManager`, which provides rate-limiting for scope as controlled by caller.

When a connection is requested, the proxy library will inform `RateLimitManager` of the event, and if the response allows a connection, it will look for the upstream with the lowest number of open connections (or in slice order if ties), increase the count, then attempt to dial the upstream. If dial fails (or when the connection is closed), the number of open connections for the upstream will be lowered (this optimistic increment lowers the number of blocking operations).

### Server

The server component of the system uses `lbproxy` to expose multiple applications, whilst providing strong security in a zero-trust network environment.
The server has the following primary responsibilities:

 1. Accepting incoming client connections
 2. Security
 3. Providing configuration to `lbproxy`

The server can be configured to proxy an arbitrary number of applications, each with an arbitrary number of upstream servers.
Each application will have an identifier (used to manage permissions), a port that the server will open to accept client connections, and a list of upstream servers, as defined in the `lbproxy` API.

`TODO: Some more details`

This simple server is not highly available, has limited capacity, and does not preserve rate limits upon restart. Some production-level improvements may include:

- Configuration should be in a centralized store.
- The state of `RateLimitManager` can be distributed using a reliable distributed cache. Since we most likely do not need exact rate-limiting, a cheaper and faster store that uses eventual consistency would be adequate. This allows rate-limits to space servers.
- We can extend the same pattern as `RateLimitManager` to the system that counts open connection per upstream, thus allowing applications to be proxied by multiple current servers.
- Once the above is in place, we can use an address-redirecting load-balancer (not a proxy load-balancer, as we would just be moving traffic up one level) to act as a gateway to our cluster of proxies. A system like Zookeeper can be used to keep a real-time tally of live servers, although of course a higher-level management backplane like Kubernetes would be more practical.

### Security

The system will provide a minimalist but robust set of security features, based on [mTLS 1.3](https://www.rfc-editor.org/rfc/rfc8446.html).

#### Transport security

The connection between server and client will be secured by mTLS. For this application, we assume there is a single CA issuing client certificates, with a manual or automated process to issue them, e.g. as part of system provisioning or licensing process. In this implementation, for simplicity, we will generate and self-sign the certificates and load them from the local file system.

- Additionally, for each upstream, we could provide a client certificate to encrypt the traffic between the proxy and the upstream (which is the server in this case), and also to ensure only authorized proxies can connect directly tp upstream servers. This may be omitted if this layer of security doesn't make sense for the application or environment in question.

#### Authentication

When a client connects with mTLS, we will inspect the name in the X509 certificate. Since we are assuming a since CA for the system, we can use the serial number as a unique client identifier. Because we also want to support authorization, this identifier is then cross-referenced to a user list, which is statically loaded for this implementation, but would be in a secure store linked to our CA/client-cert issuing system in a production implementation. If the serial number matches a known id, the client is authenticated.

#### Authorization

After validating the user id during authentication, we will load a set of claims for this user from the authorization store, which is just child data of the previous store used for authentication in this implementation. For this simple application, we will simply have a list of the allowed applications identifiers for this client.

- A simple extension to this system can include configurable rate limits for each application the client has access to, which can be tailored to usage,  application type (e.g. sockets VS HTTP connections), or licensing agreements.
- Alternatively we could have bundled claims in the X509 certificate, but that makes system maintainability and certificate management more complex.

### UX

`TODO: describe how server is operated`
