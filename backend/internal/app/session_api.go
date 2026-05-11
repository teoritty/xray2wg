package app

// SessionAPI exposes authenticated session introspection (JWT claims are parsed in middleware).
type SessionAPI struct{}

func NewSessionAPI() *SessionAPI { return &SessionAPI{} }

// MePayload is returned by GET /auth/me.
func (a *SessionAPI) MePayload(userID int64) map[string]any {
	return map[string]any{"user_id": userID}
}
