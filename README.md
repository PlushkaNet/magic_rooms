# magic_rooms
**Simple messaging tcp server one-to-all and messaging client for it writted in Go**

To build, just locate into direcory with server or client and run
```
go build main.go
```
or 
```
go build gocl.go
```

Server runs on port 10500 by default

Secure server searches for certificates (cert.pem, key.pem) in the same path as the executable

Note: tls/client currently works only with domains