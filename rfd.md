---
author: Daniele Pagano (daniele@danielepagano.com)
state: draft
---

# RFD - Basic TCP Load Balancer (Teleport Challenge)

## Context

Author is to implement a basic TCP Load Balancer (Level 4) by request of the Teleport Systems Team, as described here [here](https://github.com/gravitational/careers/blob/main/challenges/systems/challenge-2.md#level-4).

The system will be delivered as a reusable library and server (plus relevant tests) written in Go and hosted in this repository.

## Design Details

### Library: lbproxy

The core of the application is the reusable Go library `lbproxy`, which implements a configurable least-connections TCP request forwarder with a per-client rate-limiter.

At the highest level, when using lbproxy, the clients will create an instance of the `LBProxyApp`.
for each application (group of upstream server addresses) that they would like to load-balance.

When creating an `LBProxyApp`, clients will provide the following:

- An identifier for the application (string), for diagnostic purposes
- A slice of upstream servers, each a struct containing a URL and an optional client mTLS certificate (X509KeyPair).
- **TBD** other config

Once the `LBProxyApp` instance is created, it will be ready to accept client connections. Each connection will be a struct with the following details:

- Handle of underlying networking connection
- A client identifier (string)
- `TODO other config: rate limiting?`

`TODO: Describe load balancing process at a high level`

### Server

The server component of the system uses `lbproxy` to expose multiple applications, whilst providing strong security in a zero-trust environment.
The server has the following primary responsibilities:

 1. Accepting incoming client connections
 2. Security
 3. Providing configuration to `lbproxy`

The server can be configured to proxy an arbitrary number of applications, each with an arbitrary number of upstream.
Each application will have an identifier (used to manage permission), a port that server will open to accept connections, and a list of upstream servers,
as described in the library section.

`TODO: Some more details`

#### Security

We will provide a minimal but robust set of security features for this system.

- **Transport security** will be handled by mTLS 1.2 to encrypt the communication between client and proxy.
  - For each upstream, we can optionally provide a client certificate to encrypt the traffic between the proxy and the upstream. This can be omitted if the proxy and upstream are in a trusted network environment.
- **Authentication** will be keyed off the X509 certificate serial number, with each serial number corresponding to a unique client.
  - For this application, we assume there is a single CA that issues client certificates, with a manual or automated process to issue client certificates, e.g. as part of system provisioning or licensing process.
- **Authorization** will use the client id from above, and associate it to a set of claims. For this simple application, we will simply have a list of the allowed applications identifiers for this client, loaded statically. In a production system this data should be in a secure store, and may also include configurable rate limits for each client, depending on application
(i.e. sockets VS HTTP connections) and/or licensing agreements.

### UX

`TODO: describe how server is operated`
