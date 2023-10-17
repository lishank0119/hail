package hail

import (
	bytes2 "bytes"
	"errors"
	"github.com/lesismal/nbio/nbhttp/websocket"
	"net"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	Request    *http.Request
	Keys       map[string]interface{}
	keyMutex   *sync.RWMutex
	conn       *websocket.Conn
	output     chan *box
	outputDone chan struct{}
	hail       *Hail
	open       bool
	hashID     string
	rwMutex    *sync.RWMutex
	subChan    chan *box
}

func (s *Session) start(w http.ResponseWriter, r *http.Request) error {
	u := websocket.NewUpgrader()

	u.SetPongHandler(func(c *websocket.Conn, text string) {
		c.SetReadDeadline(time.Now().Add(s.hail.Option.PongWait))
		s.hail.pongHandler(s)
	})

	if s.hail.closeHandler != nil {
		u.SetCloseHandler(func(conn *websocket.Conn, i int, msg string) {
			s.hail.closeHandler(s, i, msg)
		})
	}

	u.OnOpen(func(conn *websocket.Conn) {
		s.hail.connectHandler(s)
	})

	u.OnClose(func(conn *websocket.Conn, err error) {
		if !s.hail.hub.closed() {
			s.hail.hub.unregister <- s
		}

		s.Close()
		s.hail.disconnectHandler(s)
	})

	u.OnMessage(func(c *websocket.Conn, messageType websocket.MessageType, bytes []byte) {
		c.SetReadDeadline(time.Now().Add(s.hail.Option.PongWait))

		if messageType == websocket.TextMessage {

			s.hail.messageHandler(s, bytes2.Clone(bytes))
		}

		if messageType == websocket.BinaryMessage {
			s.hail.messageHandlerBinary(s, bytes2.Clone(bytes))
		}
	})

	if s.hail.Option.CheckOrigin != nil {
		u.CheckOrigin = s.hail.Option.CheckOrigin
	}

	conn, err := u.Upgrade(w, r, w.Header())
	if err != nil {
		return err
	}

	s.conn = conn

	return nil
}

func (s *Session) Close() {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	if s.open {
		s.open = false
		s.conn.Close()
		close(s.outputDone)
		s.hail.pubSub.Unsub(s.subChan)
	}
}

func (s *Session) closed() bool {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	return !s.open
}

// IsClosed returns the status of the connection.
func (s *Session) IsClosed() bool {
	return s.closed()
}

// LocalAddr returns the local addr of the connection.
func (s *Session) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

// RemoteAddr returns the remote addr of the connection.
func (s *Session) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s *Session) writeMessage(message *box) {
	defer func() {
		if recover() != nil {
			s.hail.errorHandler(s, ErrWriteCloseSessionForRecover)
		}
	}()

	if s.closed() {
		s.hail.errorHandler(s, ErrWriteCloseSession)
		return
	}

	select {
	case s.output <- message:
	default:
		s.hail.errorHandler(s, ErrSessionMessageBufferIsFull)
	}
}

func (s *Session) writeRaw(message *box) error {
	if s.closed() {
		return ErrWriteClosed
	}

	s.conn.SetWriteDeadline(time.Now().Add(s.hail.Option.WriteWait))
	err := s.conn.WriteMessage(message.t, message.msg)

	if err != nil {
		return err
	}

	return nil
}

func (s *Session) ping() {
	s.writeRaw(&box{t: websocket.PingMessage, msg: []byte{}})
}

func (s *Session) Write(msg []byte) error {
	if s.closed() {
		return ErrWriteCloseSession
	}

	s.writeMessage(&box{t: websocket.TextMessage, msg: msg})

	return nil
}

// WriteBinary writes a binary message to session.
func (s *Session) WriteBinary(msg []byte) error {
	if s.closed() {
		return ErrWriteCloseSession
	}

	s.writeMessage(&box{t: websocket.BinaryMessage, msg: msg})
	return nil
}

// Set is used to store a new key/value pair exclusively for this session.
// It also lazy initializes s.Keys if it was not used previously.
func (s *Session) Set(key string, value interface{}) {
	s.keyMutex.Lock()
	defer s.keyMutex.Unlock()

	if s.Keys == nil {
		s.Keys = make(map[string]interface{})
	}

	s.Keys[key] = value
}

// Get returns the value for the given key, ie: (value, true).
// If the value does not exist it returns (nil, false)
func (s *Session) Get(key string) (value interface{}, exists bool) {
	s.keyMutex.RLock()
	defer s.keyMutex.RUnlock()

	if s.Keys != nil {
		value, exists = s.Keys[key]
	}

	return
}

// MustGet returns the value for the given key if it exists, otherwise it panics.
func (s *Session) MustGet(key string) interface{} {
	if value, exists := s.Get(key); exists {
		return value
	}

	panic("Key \"" + key + "\" does not exist")
}

// UnSet will delete the key and has no return value
func (s *Session) UnSet(key string) {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	if s.Keys != nil {
		delete(s.Keys, key)
	}
}

func (s *Session) GetHashID() string {
	return s.hashID
}

// AddSub 訂閱某個,多個topic (Session subscribe one or multi topics)
func (s *Session) AddSub(topicNames ...string) {
	if s.subChan != nil {
		s.hail.pubSub.AddSub(s.subChan, topicNames...)
	} else {
		var str = ""
		for _, topicName := range topicNames {
			str += topicName + ","
		}
		str = str[0 : len(str)-1]
		s.hail.errorHandler(s, errors.New("error of add current channel,"+str))
	}
}

// UnSub (Session unsubscribe one or multi topics, if no topics ,will unsubscribe all topics)
func (s *Session) UnSub(topicNames ...string) {
	if s.subChan != nil {
		s.hail.pubSub.Unsub(s.subChan, topicNames...)
	} else {
		var str = ""
		for _, topicName := range topicNames {
			str += topicName + ","
		}
		str = str[0 : len(str)-1]
		s.hail.errorHandler(s, errors.New("error of unsub current channel,"+str))
	}
}

func (s *Session) run() {
	ticker := time.NewTicker(s.hail.Option.PingPeriod)
	defer ticker.Stop()

loop:
	for {
		select {
		case msg, ok := <-s.output:
			if !ok {
				break loop
			}

			err := s.writeRaw(msg)

			if err != nil {
				s.hail.errorHandler(s, err)
				break loop
			}

			if msg.t == websocket.CloseMessage {
				break loop
			}

			if msg.t == websocket.TextMessage {
				s.hail.messageSentHandler(s, msg.msg)
			}

			if msg.t == websocket.BinaryMessage {
				s.hail.messageSentHandlerBinary(s, msg.msg)
			}
		case msg, ok := <-s.subChan:
			if !ok {
				break loop
			}
			err := s.writeRaw(msg)
			if err != nil {
				s.hail.errorHandler(s, err)
				break loop
			}

			if msg.t == websocket.CloseMessage {
				break loop
			}

			if msg.t == websocket.TextMessage {
				s.hail.messageSentHandler(s, msg.msg)
			}

			if msg.t == websocket.BinaryMessage {
				s.hail.messageSentHandlerBinary(s, msg.msg)
			}
		case <-ticker.C:
			s.ping()
		case _, ok := <-s.outputDone:
			if !ok {
				break loop
			}
		}
	}
}
