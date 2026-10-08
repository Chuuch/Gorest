package events_test

import (
	"testing"
	"time"

	"github.com/chuuch/gorest/internal/platform/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHub_PublishActivity(t *testing.T) {
	hub := events.NewHub()
	orgID := uuid.New()
	userID := uuid.New()

	ch, cancel := hub.SubscribeStaff(orgID, userID)
	defer cancel()

	hub.PublishActivity(orgID)

	select {
	case event := <-ch:
		require.Equal(t, events.ChannelActivity, event.Channel)
	case <-time.After(time.Second):
		t.Fatal("expected activity event")
	}
}

func TestHub_PublishNotification_PortalOnly(t *testing.T) {
	hub := events.NewHub()
	userID := uuid.New()

	ch, cancel := hub.SubscribePortal(userID)
	defer cancel()

	hub.PublishActivity(uuid.New())
	hub.PublishNotification(userID)

	select {
	case event := <-ch:
		require.Equal(t, events.ChannelNotification, event.Channel)
	case <-time.After(time.Second):
		t.Fatal("expected notification event")
	}

	select {
	case <-ch:
		t.Fatal("did not expect a second event")
	case <-time.After(50 * time.Millisecond):
	}
}
