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

# Server configuration
You can apply couple of different settings to server, such as port (with -p argument), mode (-m tls | none) and certificate+key (-cert /path/to/cert -key /path/to/key)

Example of starting secured server over TLS:
```
./main -p 10500 -m tls -cert /path/to/cert -key /path/to/key
```