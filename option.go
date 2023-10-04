package hail

import (
	"net/http"
	"time"
)

type Option struct {
	ChannelBufferSize int // subscribe buffer channel size
	CheckOrigin       func(r *http.Request) bool
	WriteWait         time.Duration // Milliseconds until write times out.
	PongWait          time.Duration // Timeout for waiting on pong.
	PingPeriod        time.Duration // Milliseconds between pings.
}

func (o *Option) getDefault() *Option {
	return &Option{
		WriteWait:         10 * time.Second,
		PongWait:          60 * time.Second,
		PingPeriod:        (60 * time.Second * 9) / 10,
		ChannelBufferSize: 1024 * 4,
		CheckOrigin:       nil,
	}
}

func (o *Option) reset() {
	defaultOptions := o.getDefault()

	if o.WriteWait == 0 {
		o.WriteWait = defaultOptions.WriteWait
	}

	if o.PongWait == 0 {
		o.PongWait = defaultOptions.PongWait
	}

	if o.PingPeriod == 0 {
		o.PingPeriod = defaultOptions.PingPeriod
	}

	if o.ChannelBufferSize == 0 {
		o.ChannelBufferSize = defaultOptions.ChannelBufferSize
	}

	if o.CheckOrigin == nil {
		o.CheckOrigin = defaultOptions.CheckOrigin
	}
}
