package refreshtoken

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrSessionNotFound = errors.New("refresh session not found")
	ErrSessionExpired  = errors.New("refresh session expired")
)

// Session is a server-side refresh session keyed by JTI embedded in the refresh JWT.
type Session struct {
	JTI       string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Store holds refresh sessions in memory with periodic expiry cleanup.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	stop     context.CancelFunc
	stopOnce sync.Once
}

// NewStore starts a background cleanup loop every 5 minutes until ctx is cancelled.
func NewStore(ctx context.Context) *Store {
	loopCtx, stop := context.WithCancel(ctx)
	s := &Store{
		sessions: make(map[string]*Session),
		stop:     stop,
	}
	go s.cleanupLoop(loopCtx)
	return s
}

// Stop ends the cleanup goroutine (idempotent).
func (s *Store) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.stop != nil {
			s.stop()
			s.stop = nil
		}
	})
}

func (s *Store) cleanupLoop(ctx context.Context) {
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.purgeExpired()
		}
	}
}

func (s *Store) purgeExpired() {
	now := time.Now()
	s.mu.Lock()
	for id, se := range s.sessions {
		if now.After(se.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
	s.mu.Unlock()
}

func generateJTI() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSession registers a new refresh session and returns its JTI.
func (s *Store) CreateSession(userID int64, ttl time.Duration) (jti string, err error) {
	jti, err = generateJTI()
	if err != nil {
		return "", err
	}
	now := time.Now()
	se := &Session{
		JTI:       jti,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	s.mu.Lock()
	s.sessions[jti] = se
	s.mu.Unlock()
	return jti, nil
}

// Rotate is the production-hardening "ValidateAndRevoke" operation (plan name): under a single mutex it validates
// the existing session (must exist and not be expired), deletes the old JTI, generates a new JTI,
// inserts the new session, and returns the new JTI and user id. This is atomic end-to-end for replay
// protection (concurrent refresh cannot observe a window where both JTIs are valid).
// On signing failure after Rotate, callers should treat the session as consumed; Revoke(newJTI) can roll back.
func (s *Store) Rotate(oldJTI string, ttl time.Duration) (newJTI string, userID int64, err error) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	se, ok := s.sessions[oldJTI]
	if !ok {
		return "", 0, ErrSessionNotFound
	}
	if now.After(se.ExpiresAt) {
		delete(s.sessions, oldJTI)
		return "", 0, ErrSessionExpired
	}
	delete(s.sessions, oldJTI)
	newJTI, err = generateJTI()
	if err != nil {
		return "", 0, err
	}
	s.sessions[newJTI] = &Session{
		JTI:       newJTI,
		UserID:    se.UserID,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	return newJTI, se.UserID, nil
}

// Revoke removes a session by JTI if present (idempotent).
func (s *Store) Revoke(jti string) {
	s.mu.Lock()
	delete(s.sessions, jti)
	s.mu.Unlock()
}
