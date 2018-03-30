# hades server

## Build
```
go get github.com/gorilla/mux
go get github.com/wybiral/hades/cmd/hades-server
go build github.com/wybiral/hades/cmd/hades-server
```

## Command line options
```
Usage of hades-server:
  -db string
    	specify database file (default "hades.db")
  -host string
    	specify server host (default "127.0.0.1")
  -port int
    	specify server port (default 8666)
```
