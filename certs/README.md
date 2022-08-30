# TEST CERTIFICATES - NOT FOR PRODUCTION USE

## Generation
In this sample project, we have generated sample certificates locally as follows

1. A CA certificate locally using`openssl req -newkey rsa:2048 -nodes -x509 -days 30 -out ca.crt -keyout ca.key` (plus subject details)
2. A Server key `openssl genrsa -out server.key 2048`
3. A Certificate Signing Request `openssl req -new -key server.key -days 30 -out server.csr` (plus subject details)
4. Sign CSR with `openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 30 -sha384`
5. And repeat steps 2-4 for each client certificate

## Included samples

`To be filled in during development`
