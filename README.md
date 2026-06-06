# magic_rooms
**Simple messaging tcp server one-to-all and messaging client for it writted in Go**

To build, just locate into direcory with server or client and run
```
go build
```

# Server configuration
You can apply couple of different settings to server, such as port (with -p argument), mode (-m tls | none) and certificate+key (-cert /path/to/cert -key /path/to/key)

Example of starting secured server over TLS:
```
./main -p 10500 -m tls -cert /path/to/cert -key /path/to/key
```

You can see server version before the start using:
```
./main -i v
```

# Client configuration
There are two supported command line arguments for client:

<pre>
&nbsp;-addr &lt;addr&gt;    ; to specify custom address for client to join to
&nbsp;-m &lt;tls | none&gt; ; to specify, should client use tls mode or standart insecure
</pre>

Example of usage:
```
./gocl -addr 127.0.0.1:10500 -m tls
```