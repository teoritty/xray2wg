package app

import (
	"testing"
	"time"

	"xray2wg/backend/internal/domain"
)

func TestPreserveActiveNodeOnSameSubscriptionUpdate(t *testing.T) {
	subID := int64(1)
	activeID := int64(42)
	existing := &domain.WgInterface{
		ID:             5,
		SubscriptionID: &subID,
		ActiveNodeID:   &activeID,
	}
	incoming := &domain.WgInterface{
		ID:             5,
		SubscriptionID: &subID,
		ActiveNodeID:   nil,
	}

	PreserveTunnelUpdateBindings(existing, incoming)

	if incoming.ActiveNodeID == nil || *incoming.ActiveNodeID != activeID {
		t.Fatalf("active node binding was not preserved: %#v", incoming.ActiveNodeID)
	}
}

func TestPreserveTunnelLifecycleOnMetadataUpdate(t *testing.T) {
	started := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	existing := &domain.WgInterface{
		ID:            5,
		Status:        domain.WgStatusRunning,
		ErrorMessage:  "",
		UptimeStarted: &started,
	}
	incoming := &domain.WgInterface{
		ID:            5,
		Status:        domain.WgStatusStopped,
		ErrorMessage:  "client bug",
		UptimeStarted: nil,
	}

	PreserveTunnelLifecycleOnMetadataUpdate(existing, incoming)

	if incoming.Status != domain.WgStatusRunning {
		t.Fatalf("status: got %q, want running", incoming.Status)
	}
	if incoming.ErrorMessage != "" {
		t.Fatalf("error_message: got %q, want empty", incoming.ErrorMessage)
	}
	if incoming.UptimeStarted == nil || !incoming.UptimeStarted.Equal(started) {
		t.Fatalf("uptime: got %#v, want %v", incoming.UptimeStarted, started)
	}
}

func TestPreserveActiveNodeWhenSubscriptionIDOmitted(t *testing.T) {
	subID := int64(1)
	activeID := int64(42)
	existing := &domain.WgInterface{
		ID:             5,
		SubscriptionID: &subID,
		ActiveNodeID:   &activeID,
	}
	incoming := &domain.WgInterface{
		ID:             5,
		SubscriptionID: nil,
		ActiveNodeID:   nil,
	}

	PreserveTunnelUpdateBindings(existing, incoming)

	if incoming.ActiveNodeID == nil || *incoming.ActiveNodeID != activeID {
		t.Fatalf("expected active node preserved when subscription_id omitted, got %#v", incoming.ActiveNodeID)
	}
}
