package hail

import (
	"sync"
)

type hub struct {
	sessions   map[*Session]bool
	open       bool
	rwMutex    *sync.RWMutex
	register   chan *Session
	unregister chan *Session
	broadcast  chan *box
	exit       chan *box
}

func newHub() *hub {

	return &hub{
		sessions:   make(map[*Session]bool),
		open:       true,
		rwMutex:    &sync.RWMutex{},
		register:   make(chan *Session),
		unregister: make(chan *Session),
		broadcast:  make(chan *box),
		exit:       make(chan *box),
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
		//case cs := <-h.closesession:
		//	h.rwmutex.Lock()
		//	for s := range h.sessions {
		//		if cs.keepSessionHash != "" && reflect.DeepEqual(s.hashID, cs.keepSessionHash) {
		//			continue
		//		}
		//
		//		if data, isExist := s.Get(cs.key); isExist {
		//			if reflect.DeepEqual(data, cs.value) {
		//				s.Close()
		//				break
		//			}
		//		}
		//	}
		//	h.rwmutex.Unlock()
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
