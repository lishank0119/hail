package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lishank0119/hail"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	flag.Parse()

	h := hail.New(&hail.Option{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	})

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
