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

// AuditPersister is implemented by the DB audit log repo.
type AuditPersister interface {
	Save(level, source, msg string) error
}

type EventLog struct {
	mu     sync.Mutex
	max    int
	events []EventEntry
	db     AuditPersister
}

func NewEventLog(max int) *EventLog {
	if max <= 0 {
		max = 64
	}
	return &EventLog{max: max}
}

// SetPersister attaches a DB backend; called once after DB init.
func (e *EventLog) SetPersister(p AuditPersister) {
	e.mu.Lock()
	e.db = p
	e.mu.Unlock()
}

func (e *EventLog) Add(level, msg string) {
	e.AddSource(level, "system", msg)
}

func (e *EventLog) AddSource(level, source, msg string) {
	e.mu.Lock()
	e.events = append(e.events, EventEntry{TS: time.Now().UTC(), Level: level, Message: msg})
	if len(e.events) > e.max {
		e.events = e.events[len(e.events)-e.max:]
	}
	p := e.db
	e.mu.Unlock()

	if p != nil && (level == "warn" || level == "error") {
		_ = p.Save(level, source, msg)
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
