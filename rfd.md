---
author: Daniele Pagano (daniele@danielepagano.com)
state: draft
---

# RFD - Basic TCP Load Balancer (Teleport Challenge)

## What & Why

Author is to implement a basic TCP Load Balancer (Level 4) by request of the Teleport Systems Team, as described [here](https://github.com/gravitational/careers/blob/main/challenges/systems/challenge-2.md#level-4).

The system will be delivered as a reusable library and server (plus relevant tests) written in Go and hosted in this repository.

## Details

### Library: lbproxy

The core of the application is the reusable Go library `lbproxy`, which implements a configurable least-connections TCP request forwarder with a per-client rate-limiter.

At the highest level, when using lbproxy, the clients will create an instance of the `LBProxyApp`.
for each application (group of upstream server addresses) that they would like to load-balance.

When creating an `LBProxyApp`, clients will provide the following:

- An identifier for the application (string), for diagnostic purposes
- A slice of upstream servers, each a struct containing a URL and an optional client mTLS certificate (X509KeyPair).
- `TBD other config`

Once the `LBProxyApp` instance is created, it will be ready to accept client connections. Each connection will be a struct with the following details:

- Handle of underlying networking connection
- A client identifier (string)
- `TODO other config: rate limiting?`

`TODO: Split overview and API. Describe load balancing process at a high level`

### Server

The server component of the system uses `lbproxy` to expose multiple applications, whilst providing strong security in a zero-trust network environment.
The server has the following primary responsibilities:

 1. Accepting incoming client connections
 2. Security
 3. Providing configuration to `lbproxy`

The server can be configured to proxy an arbitrary number of applications, each with an arbitrary number of upstream servers.
Each application will have an identifier (used to manage permissions), a port that the server will open to accept client connections, and a list of upstream servers, as defined in the `lbproxy` API.

`TODO: Some more details`

#### Security

The system will provide a minimal but robust set of security features.

- **Transport security** will be handled by mTLS 1.2 to encrypt the communication between client and proxy.
  - Additionally, for each upstream, we may provide a client certificate to encrypt the traffic between the proxy and the upstream (which is the server in this case), and also to ensure only authorized proxies can connect directly tp upstream servers. This may be omitted if this layer of security doesn't make sense for the application or environment in question.
- **Authentication** will be keyed off the X509 certificate serial number, with each serial number corresponding to a unique client.
  - For this application, we assume there is a single CA that issues client certificates, with a manual or automated process to issue these certificates, e.g. as part of system provisioning or licensing process. We will also simply look for certificates on the local file system.
- **Authorization** will use the client id from above, and associate it to a set of claims. For this simple application, we will simply have a list of the allowed applications identifiers for this client, loaded statically. In a production system this data should be in a secure store, and may also include configurable rate limits for each client, depending on the specific application (e.g. sockets VS HTTP connections) and/or licensing agreements.

### UX

`TODO: describe how server is operated`
