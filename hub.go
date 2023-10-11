package hail

import (
	"sync"
	"time"
)

type hub struct {
	Option       *Option
	sessions     map[*Session]bool
	open         bool
	rwMutex      *sync.RWMutex
	register     chan *Session
	unregister   chan *Session
	broadcast    chan *box
	exit         chan *box
	closeSession chan *box
}

func newHub(o *Option) *hub {
	return &hub{
		Option:       o,
		sessions:     make(map[*Session]bool),
		open:         true,
		rwMutex:      &sync.RWMutex{},
		register:     make(chan *Session),
		unregister:   make(chan *Session),
		broadcast:    make(chan *box),
		exit:         make(chan *box),
		closeSession: make(chan *box),
	}
}

func (h *hub) closed() bool {
	h.rwMutex.RLock()
	defer h.rwMutex.RUnlock()
	return !h.open
}

func (h *hub) run() {
loop:
	for {
		select {
		case s := <-h.register:
			h.rwMutex.Lock()
			h.sessions[s] = true
			h.rwMutex.Unlock()
		case s := <-h.unregister:
			if _, ok := h.sessions[s]; ok {
				h.rwMutex.Lock()
				delete(h.sessions, s)
				h.rwMutex.Unlock()
			}
		case m := <-h.closeSession:
			h.rwMutex.Lock()
			for s := range h.sessions {
				if m.filter != nil {
					if m.filter(s) {
						s.writeMessage(m)
						time.AfterFunc(h.Option.CloseSessionWaitTime, func() {
							s.Close()
						})
					}
				} else {
					s.writeMessage(m)
					time.AfterFunc(h.Option.CloseSessionWaitTime, func() {
						s.Close()
					})
				}
			}
			h.rwMutex.Unlock()
		case m := <-h.broadcast:
			h.rwMutex.RLock()
			for s := range h.sessions {
				if m.filter != nil {
					if m.filter(s) {
						s.writeMessage(m)
					}
				} else {
					s.writeMessage(m)
				}
			}
			h.rwMutex.RUnlock()
		case m := <-h.exit:
			h.rwMutex.Lock()
			for s := range h.sessions {
				s.writeMessage(m)
				delete(h.sessions, s)
				s.Close()
			}
			h.open = false
			h.rwMutex.Unlock()
			break loop
		}
	}
}
