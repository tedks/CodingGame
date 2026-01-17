package debug

import "time"

// EventType represents the type of debug event.
type EventType int

const (
	// EventStateChanged indicates the session state changed.
	EventStateChanged EventType = iota
	// EventStackChanged indicates the call stack changed.
	EventStackChanged
	// EventBreakpointHit indicates a breakpoint was hit.
	EventBreakpointHit
	// EventBreakpointSet indicates a breakpoint was added.
	EventBreakpointSet
	// EventBreakpointRemoved indicates a breakpoint was removed.
	EventBreakpointRemoved
	// EventStep indicates a step operation completed.
	EventStep
	// EventDataFlow indicates a data flow event for belt visualization.
	EventDataFlow
	// EventOutput indicates debugger output (stdout/stderr).
	EventOutput
	// EventError indicates an error occurred.
	EventError
)

// String returns a human-readable event type name.
func (et EventType) String() string {
	switch et {
	case EventStateChanged:
		return "StateChanged"
	case EventStackChanged:
		return "StackChanged"
	case EventBreakpointHit:
		return "BreakpointHit"
	case EventBreakpointSet:
		return "BreakpointSet"
	case EventBreakpointRemoved:
		return "BreakpointRemoved"
	case EventStep:
		return "Step"
	case EventDataFlow:
		return "DataFlow"
	case EventOutput:
		return "Output"
	case EventError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Event represents a debug event.
type Event struct {
	Type      EventType              // Event type
	SessionID string                 // Session this event belongs to
	Timestamp time.Time              // When this event occurred
	Data      map[string]interface{} // Event-specific data
}

// NewEvent creates a new debug event.
func NewEvent(eventType EventType, sessionID string) *Event {
	return &Event{
		Type:      eventType,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}
}

// WithData adds data to the event.
func (e *Event) WithData(key string, value interface{}) *Event {
	e.Data[key] = value
	return e
}

// GetString retrieves a string value from event data.
func (e *Event) GetString(key string) string {
	if v, ok := e.Data[key].(string); ok {
		return v
	}
	return ""
}

// GetInt retrieves an int value from event data.
func (e *Event) GetInt(key string) int {
	if v, ok := e.Data[key].(int); ok {
		return v
	}
	if v, ok := e.Data[key].(float64); ok {
		return int(v)
	}
	return 0
}

// GetBool retrieves a bool value from event data.
func (e *Event) GetBool(key string) bool {
	if v, ok := e.Data[key].(bool); ok {
		return v
	}
	return false
}

// EventHandler is a function that handles debug events.
type EventHandler func(*Event)

// EventBus manages debug event distribution.
type EventBus struct {
	handlers []EventHandler
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make([]EventHandler, 0),
	}
}

// Subscribe adds an event handler.
func (bus *EventBus) Subscribe(handler EventHandler) {
	bus.handlers = append(bus.handlers, handler)
}

// Publish sends an event to all handlers.
func (bus *EventBus) Publish(event *Event) {
	for _, h := range bus.handlers {
		h(event)
	}
}
