# TEST CERTIFICATES - NOT FOR PRODUCTION USE

## Generation
In this sample project, we have generated sample certificates locally as follows

1. A CA certificate locally using `openssl req -newkey rsa:2048 -nodes -x509 -days 30 -out ca.crt -keyout ca.key` (plus subject details)
2. A Server key `openssl genrsa -out server.key 2048`
3. A Certificate Signing Request `openssl req -new -key server.key -days 30 -out server.csr` (plus subject details)
4. Sign CSR with `openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 30 -sha384`
5. And repeat steps 2-4 for each client certificate, replacing 'server' with the name of the client you need

## Included samples

`NOTE: Included files are only valid for Sept 2022`

### CA and server

These are located in the `certs/` directory

- CA certs: `ca.crt` and `ca.key` from step #1, issued to CA `localhost`
- `server.key` from step #2 using `openssl req -new -key server.key -days 30 -out server.csr -subj "/C=US/ST=UT/L=SLC/O=Tets/CN=*.localhost"`
- `server.csr` from step #3 using `openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 30 -sha384 -extfile <(printf "subjectAltName=DNS:localhost")`
- `server.crt` from step $4, valid for Sept 2022

### Client certs for "localhost"

These are under the `certs/clients` directory, plus a directory for the client CN for auto-loading. 
We use a simple convention in the server where the client id is the file name,
for example for `CN=localhost` you would use `certs/clients/localhost/localhost.crt` etc.

From `certs` folder

1. `openssl genrsa -out clients/localhost/localhost.key 2048`
2. `openssl req -new -key clients/localhost/localhost.key -days 30 -out clients/localhost/localhost.csr -subj "/C=US/ST=UT/L=SLC/O=Tets/CN=localhost"`
3. `openssl x509 -req -in clients/localhost/localhost.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out clients/localhost/localhost.crt -days 30 -sha384 -extfile <(printf "subjectAltName=DNS:localhost")`

Simple test from command line
```shell
curl -k --cacert ca.crt \
	--cert clients/localhost/localhost.crt \
	--key clients/localhost/localhost.key \
	https://localhost:9001
```