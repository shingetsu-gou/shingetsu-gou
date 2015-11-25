[![Build Status](https://travis-ci.org/shingetsu-gou/http-relay.svg?branch=master)](https://travis-ci.org/shingetsu-gou/http-relay)
[![GoDoc](https://godoc.org/github.com/shingetsu-gou/http-relay?status.svg)](https://godoc.org/github.com/shingetsu-gou/http-relay)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/shingetsu-gou/http-relay/master/LICENSE)


# http-relay 

## Overview

http-relay is a module for relaying http by websocket in golang.

When node B is behind NAT and node C wants to connect to B (i.e. needs NAT traversal),

0. B requests relay server A to relay by websocket
1. C connects to the relay server A with http
2. A relays data to B with websocket
3. B responses to A with websocket
4. A relays B's response to C with http 

i.e. C  <-http->  A  <-websocket->  B

## Requirements

* git
* go 1.4+

are required to compile.

## How to Get

    $ go get github.com/shingetsu-gou/http-relay

## Example

Suppose C wants to communicate with B by relaying A, 
C  <-http->  A  <-websocket->  B:

```go
//relay server A
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		relay.HandleServer("test", w, r, func(r *ResponseWriter) bool {
			//You can check resopnse from relay client B
			return true
		})
	})
	http.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		relay.Serve("test", ws)
	}))
	http.ListenAndServe(":1234", nil)
	
//relay client B
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world!"))
	})
	origin := "http://localhost/"
	url := "ws://localhost:1234/ws"
	closed:=make(chan struct{}) //channel for signaling websocket was closed
	relay.HandleClient(url, origin, http.DefaultServeMux, closed,func(r *http.Request) {
		r.URL.Path = "/hello"
	})

//node that want to connect relay client C
	res, _:= http.Get("http://localhost:1234/")
	body, _:= ioutil.ReadAll(res.Body)
    res.Body.Close()
	//body must be "hello world!"

```

## License

MIT License

# Contribution

Improvements to the codebase and pull requests are encouraged.
