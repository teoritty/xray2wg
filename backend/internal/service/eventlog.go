package service

import (
	"sync"
	"time"
)

type EventEntry struct {
	TS      time.Time `json:"ts"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

type EventLog struct {
	mu     sync.Mutex
	max    int
	events []EventEntry
}

func NewEventLog(max int) *EventLog {
	if max <= 0 {
		max = 64
	}
	return &EventLog{max: max}
}

func (e *EventLog) Add(level, msg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, EventEntry{TS: time.Now().UTC(), Level: level, Message: msg})
	if len(e.events) > e.max {
		e.events = e.events[len(e.events)-e.max:]
	}
}

func (e *EventLog) List() []EventEntry {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := make([]EventEntry, len(e.events))
	copy(cp, e.events)
	for i, j := 0, len(cp)-1; i < j; i, j = i+1, j-1 {
		cp[i], cp[j] = cp[j], cp[i]
	}
	return cp
}
