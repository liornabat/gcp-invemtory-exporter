package logger

import (
	"sync"
)

type bufferSink struct {
	sync.Mutex
	buffer  []string
	enabled bool
}

func newBufferSink() *bufferSink {
	return &bufferSink{
		buffer:  []string{},
		enabled: false,
	}
}

func (s *bufferSink) setEnabled(enabled bool) *bufferSink {
	s.Lock()
	defer s.Unlock()
	s.enabled = enabled
	return s
}

func (s *bufferSink) Print(msg string) {
	s.Lock()
	defer s.Unlock()
	if s.enabled {
		s.buffer = append(s.buffer, msg)
	}
}

func (s *bufferSink) getBufferLogs() []string {
	s.Lock()
	defer s.Unlock()
	var logs []string
	logs = append(logs, s.buffer...)
	s.buffer = []string{}
	return logs
}

func (s *bufferSink) Sync() error {
	return nil
}
