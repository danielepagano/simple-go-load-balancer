# TEST CERTIFICATES - NOT FOR PRODUCTION USE

## Generation
In this sample project, we have generated sample certificates locally as follows

1. A CA certificate locally using`openssl req -newkey rsa:2048 -nodes -x509 -days 30 -out ca.crt -keyout ca.key` (plus subject details)
2. A Server key `openssl genrsa -out server.key 2048`
3. A Certificate Signing Request `openssl req -new -key server.key -days 30 -out server.csr` (plus subject details)
4. Sign CSR with `openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 30 -sha384`
5. And repeat steps 2-4 for each client certificate, replacing 'server' with the name of the client you need

## Included samples

`NOTE: Included files are only valid for Sept 2022`

### CA and server

These are located in the `certs/` directory

- CA certs: `ca.crt` and `ca.key` from step #1, issued to CA `danielepagano.com`
- `server.key` from step #2
- `server.csr` from step #3
  - CN=`danielepagano.com`, challenge password `5TA&K#&nWE4oIcQf`
- `server.crt` from step $4, valid for Sept 2022

### Client certs for "localhost"

These are under the `certs/clients` directory, plus a directory for the client CN for auto-loading. 
We use a simple convention in the server where the client id is the file name,
for example for `CN=localhost` you would use `certs/clients/localhost/localhost.crt` etc.

1. `openssl genrsa -out localhost.key 2048`
2. `openssl req -new -key localhost.key -days 30 -out localhost.csr` 
   - CN=`localhost`, challenge password `EQmvv^BY$L1n9Pa#`
3. `openssl x509 -req -in localhost.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out localhost.crt -days 30 -sha384`

Simple test from command line
```shell
curl --trace trace.log -k \
	--cacert ./certs/ca.crt \
	--cert ./certs/clients/localhost/localhost.crt \
	--key ./certs/clients/localhost/localhost.key \
	https://localhost:9001
```