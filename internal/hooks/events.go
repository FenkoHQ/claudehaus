package hooks

import (
	"sync"
	"time"
)

type Event struct {
	ID         string
	SessionID  string
	Timestamp  string
	EventName  string
	ToolName   string
	ToolInput  string
	Detail     string
}

type EventStore struct {
	mu     sync.RWMutex
	events []Event
}

func NewEventStore() *EventStore {
	return &EventStore{
		events: make([]Event, 0, 100),
	}
}

func (s *EventStore) Add(event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)

	// Keep only the last 500 events per session
	if len(s.events) > 500 {
		// Remove oldest events for the same session
		count := 0
		newEvents := make([]Event, 0, len(s.events)-1)
		for _, e := range s.events {
			if e.SessionID != event.SessionID || count > 0 {
				newEvents = append(newEvents, e)
			} else {
				count++
			}
		}
		s.events = newEvents
	}
}

func (s *EventStore) GetBySession(sessionID string, limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Event, 0)
	for i := len(s.events) - 1; i >= 0; i-- {
		if s.events[i].SessionID == sessionID {
			result = append(result, s.events[i])
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

func (s *EventStore) AddEvent(sessionID, eventName, toolName, toolInput, detail string) {
	s.Add(Event{
		ID:        generateEventID(),
		SessionID: sessionID,
		Timestamp: time.Now().Format("15:04:05"),
		EventName: eventName,
		ToolName:  toolName,
		ToolInput: toolInput,
		Detail:    detail,
	})
}

func generateEventID() string {
	return time.Now().Format("20060102150405.999999999")
}
