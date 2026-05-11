package wshub

import (
	"context"
	"testing"
	"time"
)

func TestHubShutdownIdempotent(t *testing.T) {
	h := NewHub(context.Background())
	h.Shutdown()
	h.Shutdown()
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHubParentCancelClosesRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := NewHub(ctx)
	cancel()
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHubCloseWaitsRun(t *testing.T) {
	h := NewHub(context.Background())
	done := make(chan struct{})
	go func() {
		_ = h.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return")
	}
}

func TestHub_Shutdown_ClosesRunGracefully(t *testing.T) {
	TestHubCloseWaitsRun(t)
}

func TestHub_Shutdown_Idempotent(t *testing.T) {
	TestHubShutdownIdempotent(t)
}

func TestHub_ContextCancelled(t *testing.T) {
	TestHubParentCancelClosesRun(t)
}

func TestHubRunReturnsAfterClose(t *testing.T) {
	h := NewHub(context.Background())
	done := make(chan struct{})
	go func() {
		h.Run()
		close(done)
	}()
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after Close")
	}
}
