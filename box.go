package hail

import "github.com/lesismal/nbio/nbhttp/websocket"

type box struct {
	t      websocket.MessageType
	msg    []byte
	filter filterFunc
}

type closeSession struct {
	t               websocket.MessageType
	key             string
	value           interface{}
	keepSessionHash string
}
