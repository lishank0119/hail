# Hail
Hail is websocket framework based on [nbio](https://github.com/lesismal/nbio) and [melody](https://github.com/olahol/melody) that abstracts away the tedious parts of handling websockets. It gets out of your way so you can write real-time apps.

## Features
* [x] Clear and easy interface similar to nbio.
* [x] A simple way to broadcast to all or selected connected sessions.
* [x] Message buffers making concurrent writing safe.
* [x] Automatic handling of sending ping/pong heartbeats that timeout broken sessions.
* [x] Store data on sessions.
* [ ] Pub/Sub.
* [ ] close some sessions.


## Install
```
go get github.com/lishank0119/hail
```

## Example

```go

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/lesismal/nbio/nbhttp"
	"hail"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	flag.Parse()

	h := hail.New(&hail.Option{})

	mux := &http.ServeMux{}
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		sessionDataMap := make(map[string]interface{}, 0)
		sessionDataMap["ip"] = r.RemoteAddr
		h.AddConnect(w, r, sessionDataMap)
	})

	h.HandleConnect(func(session *hail.Session) {
		fmt.Println("HandleConnect", session.GetHashID())
	})

	h.HandleMessage(func(session *hail.Session, bytes []byte) {
		session.Write(bytes)

		fmt.Println("HandleMessage", session.GetHashID(), string(bytes))
	})

	h.HandleClose(func(session *hail.Session, err error) {
		fmt.Println("HandleClose", session.GetHashID())
	})

	h.HandleMessageBinary(func(session *hail.Session, bytes []byte) {
		session.WriteBinary(bytes)
		fmt.Println("HandleMessageBinary", session.GetHashID(), len(bytes))
	})

	h.HandleDisconnect(func(session *hail.Session) {
		fmt.Println("HandleDisconnect", session.GetHashID())
	})

	svr := nbhttp.NewServer(nbhttp.Config{
		Network:                 "tcp",
		Addrs:                   addrs,
		MaxLoad:                 1000000,
		ReleaseWebsocketPayload: true,
		Handler:                 mux,
		ReadBufferSize:          1024 * 4,
		IOMod:                   nbhttp.IOModMixed,
		MaxBlockingOnline:       100000,
	})

	err := svr.Start()
	if err != nil {
		fmt.Printf("nbio.Start failed: %v\n", err)
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	svr.Shutdown(ctx)
}

var addrs = []string{
	":8888",
}

```

## Contributors
<a href="https://github.com/lishank0119/hail/graphs/contributors">
	<img src="https://contrib.rocks/image?repo=lishank0119/hail" />
</a>


## FAQ

If you are getting a `403` when trying  to connect to your websocket you can [change allow all origin hosts](http://godoc.org/github.com/gorilla/websocket#hdr-Origin_Considerations):

```go
hail.New(&hail.Option{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
})
```

<a href='https://ko-fi.com/Z8Z7GSFJE' target='_blank'><img height='36' style='border:0px;height:36px;' src='https://storage.ko-fi.com/cdn/kofi2.png?v=3' border='0' alt='Buy Me a Coffee at ko-fi.com' /></a>