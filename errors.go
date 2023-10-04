package hail

import "errors"

var (
	ErrWriteCloseSessionForRecover = errors.New("tried to write to closed a session for recover")
	ErrWriteCloseSession           = errors.New("tried to write to closed a session")
	ErrSessionMessageBufferIsFull  = errors.New("session message buffer is full")
	ErrHubClose                    = errors.New("hail hub is closed")
	ErrClose                       = errors.New("hail instance is closed")
	ErrWriteClosed                 = errors.New("tried to write to closed a session")
)
