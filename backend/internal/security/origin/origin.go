package origin

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// Config holds a set of allowed browser origins (scheme + host + optional port).
// Matching is exact only after normalization (lowercase scheme and host; port preserved).
type Config struct {
	mu      sync.RWMutex
	allowed map[string]struct{}
}

// NewConfig parses a comma-separated list of origins. Each entry must be a valid origin URL
// with scheme and host. Returns an error if allowedList is empty or yields no valid origins,
// or if any non-empty entry is invalid.
func NewConfig(allowedList string) (*Config, error) {
	parts := strings.Split(allowedList, ",")
	allowed := make(map[string]struct{})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		norm, err := normalizeOrigin(part)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed origin %q: %w", part, err)
		}
		allowed[norm] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS is required and must list at least one valid origin")
	}
	return &Config{allowed: allowed}, nil
}

// AllowOrigin reports whether origin is allowed (exact match after normalization).
// Empty origin is rejected.
func (c *Config) AllowOrigin(origin string) bool {
	if strings.TrimSpace(origin) == "" {
		return false
	}
	norm, err := normalizeOrigin(origin)
	if err != nil {
		return false
	}
	c.mu.RLock()
	_, ok := c.allowed[norm]
	c.mu.RUnlock()
	return ok
}

// AllowedOrigins returns a sorted copy of configured origins.
func (c *Config) AllowedOrigins() []string {
	c.mu.RLock()
	out := make([]string, 0, len(c.allowed))
	for o := range c.allowed {
		out = append(out, o)
	}
	c.mu.RUnlock()
	sort.Strings(out)
	return out
}

func normalizeOrigin(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty origin")
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("origin must include scheme and host")
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Host)
	return scheme + "://" + host, nil
}
