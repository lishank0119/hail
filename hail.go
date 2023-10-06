package hail

import (
	"github.com/google/uuid"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"net/http"
	"sync"
)

type handleMessageFunc func(*Session, []byte)
type handleErrorFunc func(*Session, error)
type handleCloseFunc func(*Session, error)
type handleSessionFunc func(*Session)
type filterFunc func(*Session) bool

type Hail struct {
	Option                   *Option
	messageHandler           handleMessageFunc
	messageHandlerBinary     handleMessageFunc
	messageSentHandler       handleMessageFunc
	messageSentHandlerBinary handleMessageFunc
	errorHandler             handleErrorFunc
	closeHandler             handleCloseFunc
	connectHandler           handleSessionFunc
	disconnectHandler        handleSessionFunc
	pongHandler              handleSessionFunc
	hub                      *hub
}

func New(o *Option) *Hail {
	o.reset()

	hub := newHub()

	go hub.run()

	return &Hail{
		Option:                   o,
		messageHandler:           func(*Session, []byte) {},
		messageHandlerBinary:     func(*Session, []byte) {},
		messageSentHandler:       func(*Session, []byte) {},
		messageSentHandlerBinary: func(*Session, []byte) {},
		errorHandler:             func(*Session, error) {},
		closeHandler:             nil,
		connectHandler:           func(*Session) {},
		disconnectHandler:        func(*Session) {},
		pongHandler:              func(*Session) {},
		hub:                      hub,
	}

}

// HandleConnect fires fn when a session connects.
func (h *Hail) HandleConnect(fn func(*Session)) {
	h.connectHandler = fn
}

// HandleDisconnect fires fn when a session disconnects.
func (h *Hail) HandleDisconnect(fn func(*Session)) {
	h.disconnectHandler = fn
}

// HandlePong fires fn when a pong is received from a session.
func (h *Hail) HandlePong(fn func(*Session)) {
	h.pongHandler = fn
}

// HandleMessage fires fn when a text message comes in.
func (h *Hail) HandleMessage(fn func(*Session, []byte)) {
	h.messageHandler = fn
}

// HandleMessageBinary fires fn when a binary message comes in.
func (h *Hail) HandleMessageBinary(fn func(*Session, []byte)) {
	h.messageHandlerBinary = fn
}

// HandleSentMessage fires fn when a text message is successfully sent.
func (h *Hail) HandleSentMessage(fn func(*Session, []byte)) {
	h.messageSentHandler = fn
}

// HandleSentMessageBinary fires fn when a binary message is successfully sent.
func (h *Hail) HandleSentMessageBinary(fn func(*Session, []byte)) {
	h.messageSentHandlerBinary = fn
}

// HandleError fires fn when a session has an error.
func (h *Hail) HandleError(fn func(*Session, error)) {
	h.errorHandler = fn
}

func (h *Hail) HandleClose(fn func(*Session, error)) {
	if fn != nil {
		h.closeHandler = fn
	}
}

func (h *Hail) AddConnect(w http.ResponseWriter, r *http.Request, keys map[string]interface{}) error {
	if h.hub.closed() {
		return ErrHubClose
	}

	session := &Session{
		Request:    r,
		Keys:       keys,
		output:     make(chan *box, h.Option.ChannelBufferSize),
		outputDone: make(chan struct{}), // fix write to close output channel
		hail:       h,
		open:       true,
		rwMutex:    &sync.RWMutex{},
		keyMutex:   &sync.RWMutex{},
		hashID:     uuid.NewString(),
	}

	err := session.start(w, r)
	if err != nil {
		return err
	}

	h.hub.register <- session

	go session.run()

	return nil
}

func (h *Hail) Broadcast(msg []byte) error {
	if h.hub.closed() {
		return ErrClose
	}

	message := &box{t: websocket.TextMessage, msg: msg}
	h.hub.broadcast <- message

	return nil
}

// BroadcastFilter broadcasts a text message to all sessions that fn returns true for.
func (h *Hail) BroadcastFilter(msg []byte, fn func(*Session) bool) error {
	if h.hub.closed() {
		return ErrClose
	}

	message := &box{t: websocket.TextMessage, msg: msg, filter: fn}
	h.hub.broadcast <- message

	return nil
}

// BroadcastBinary broadcasts a binary message to all sessions.
func (h *Hail) BroadcastBinary(msg []byte) error {
	if h.hub.closed() {
		return ErrClose
	}

	message := &box{t: websocket.BinaryMessage, msg: msg}
	h.hub.broadcast <- message

	return nil
}

// BroadcastBinaryFilter broadcasts a binary message to all sessions that fn returns true for.
func (h *Hail) BroadcastBinaryFilter(msg []byte, fn func(*Session) bool) error {
	if h.hub.closed() {
		return ErrClose
	}

	message := &box{t: websocket.BinaryMessage, msg: msg, filter: fn}
	h.hub.broadcast <- message

	return nil
}
