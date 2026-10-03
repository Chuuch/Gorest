package events

import (
	"sync"

	"github.com/google/uuid"
)

const ChannelActivity = "activity"
const ChannelNotification = "notification"

type Event struct {
	Channel string `json:"channel"`
}

type Hub struct {
	mu   sync.RWMutex
	subs map[chan Event]*subscription
}

type subscription struct {
	organizationID uuid.UUID
	userID         uuid.UUID
	staff          bool
}

func NewHub() *Hub {
	return &Hub{subs: make(map[chan Event]*subscription)}
}

func (h *Hub) SubscribeStaff(organizationID, userID uuid.UUID) (<-chan Event, func()) {
	return h.subscribe(organizationID, userID, true)
}

func (h *Hub) SubscribePortal(userID uuid.UUID) (<-chan Event, func()) {
	return h.subscribe(uuid.Nil, userID, false)
}

func (h *Hub) subscribe(
	organizationID, userID uuid.UUID,
	staff bool,
) (<-chan Event, func()) {
	ch := make(chan Event, 8)

	h.mu.Lock()
	h.subs[ch] = &subscription{
		organizationID: organizationID,
		userID:         userID,
		staff:          staff,
	}
	h.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs, ch)
			h.mu.Unlock()
			close(ch)
		})
	}

	return ch, cancel
}

func (h *Hub) PublishActivity(organizationID uuid.UUID) {
	h.publish(Event{Channel: ChannelActivity}, func(sub *subscription) bool {
		return sub.staff && sub.organizationID == organizationID
	})
}

func (h *Hub) PublishNotification(userID uuid.UUID) {
	h.publish(Event{Channel: ChannelNotification}, func(sub *subscription) bool {
		return sub.userID == userID
	})
}

func (h *Hub) publish(event Event, match func(*subscription) bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch, sub := range h.subs {
		if !match(sub) {
			continue
		}
		select {
		case ch <- event:
		default:
		}
	}
}
