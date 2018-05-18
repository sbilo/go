package sse

import (
	"context"
	"net/http"
	"time"
)

// Stream represents an output stream that data can be written to
type Stream interface {
	TrySendHeartbeat()
	Send(Event)
	SentCount() int
	Finish()
	SetLimit(limit int)
	IsDone() bool
	Err(error)
	IsEstablished() bool
}

// NewStream creates a new stream against the provided response writer
func NewStream(ctx context.Context, w http.ResponseWriter, r *http.Request) Stream {
	result := &stream{ctx, w, r, false, 0, 0, time.Now(), false}
	return result
}

type stream struct {
	ctx          context.Context
	w            http.ResponseWriter
	r            *http.Request
	done         bool
	sent         int
	limit        int
	lastWriteAt  time.Time
	sentPreamble bool
}

func (s *stream) Send(e Event) {

	s.tryWritePreamble()
	WriteEvent(s.ctx, s.w, e)
	s.lastWriteAt = time.Now()
	s.sent++
}

func (s *stream) TrySendHeartbeat() {

	if time.Since(s.lastWriteAt) < HeartbeatDelay {
		return
	}

	s.tryWritePreamble()
	WriteHeartbeat(s.ctx, s.w)
	s.lastWriteAt = time.Now()
}

func (s *stream) SentCount() int {
	return s.sent
}

func (s *stream) SetLimit(limit int) {
	s.limit = limit
}

func (s *stream) Finish() {
	s.tryWritePreamble()
	WriteEvent(s.ctx, s.w, goodbyeEvent)
	s.done = true
}

func (s *stream) IsDone() bool {
	if s.limit == 0 {
		return s.done
	}

	return s.done || s.sent >= s.limit
}

func (s *stream) Err(err error) {
	WriteEvent(s.ctx, s.w, Event{Error: err})
	s.done = true
}

func (s *stream) IsEstablished() bool {
	return s.sentPreamble
}

func (s *stream) tryWritePreamble() {
	if s.sentPreamble {
		return
	}

	ok := WritePreamble(s.ctx, s.w)
	if !ok {
		s.done = true
		return
	}

	s.sentPreamble = true
}
