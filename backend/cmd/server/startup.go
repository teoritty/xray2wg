package main

import (
	"context"
	"database/sql"
	"strings"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/service"

	"github.com/rs/zerolog/log"
)

// subscriptionLooper is the subset of SubscriptionService used at process startup (testable).
type subscriptionLooper interface {
	StartRefreshLoop(ctx context.Context, id int64)
}

// startSubscriptionLoops starts refresh workers for non-manual subscriptions (skips inactive / empty URL).
func startSubscriptionLoops(ctx context.Context, subs []*domain.Subscription, subSvc subscriptionLooper) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if subSvc == nil {
		return nil
	}
	for _, s := range subs {
		if s == nil {
			continue
		}
		if s.Name == service.ManualSubscriptionName {
			continue
		}
		if s.Status == domain.SubStatusInactive {
			continue
		}
		if strings.TrimSpace(s.URL) == "" {
			continue
		}
		subSvc.StartRefreshLoop(ctx, s.ID)
	}
	return nil
}

// logTeardownError logs database close failures during process shutdown (production hardening: never ignore _ on Close).
func logTeardownError(db *sql.DB) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		log.Error().Err(err).Msg("sql db close")
	}
}
