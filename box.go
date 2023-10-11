package hail

import "github.com/lesismal/nbio/nbhttp/websocket"

type box struct {
	t      websocket.MessageType
	msg    []byte
	filter filterFunc
}
