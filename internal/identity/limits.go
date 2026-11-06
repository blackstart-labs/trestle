package identity

import (
	"strings"
	"sync"
	"time"
)

// In-memory login rate limiter (identity-policy.md):
//   - IP + canonical username: 5 failures per 15 minutes.
//   - IP across usernames: 30 failures per 15 minutes.
//   - No permanent lockout; a correct password does not bypass an active limit.
//   - Uses the TCP peer IP; never trusts X-Forwarded-For.
const (
	limitWindow = 15 * 60 // seconds
	ipUserLimit = 5
	ipLimit     = 30
	maxBuckets  = 10_000
)

type bucket struct {
	count       uint32
	windowStart int64
}

func (b *bucket) recordFailure(now int64) uint32 {
	if now-b.windowStart >= limitWindow {
		b.windowStart, b.count = now, 0
	}
	b.count++
	return b.count
}

func (b *bucket) current(now int64) uint32 {
	if now-b.windowStart >= limitWindow {
		return 0
	}
	return b.count
}

type LoginLimiter struct {
	mu     sync.Mutex
	ipUser map[[2]string]*bucket
	ipOnly map[string]*bucket
	now    func() int64
}

func NewLoginLimiter() *LoginLimiter {
	return &LoginLimiter{
		ipUser: map[[2]string]*bucket{},
		ipOnly: map[string]*bucket{},
		now:    func() int64 { return time.Now().Unix() },
	}
}

// RecordFailure records a failed login; it returns true when the limit is now exceeded.
func (l *LoginLimiter) RecordFailure(ip, username string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	// Bound memory under a spray of IPs or usernames; amortised, as it runs only past the cap.
	if len(l.ipUser) > maxBuckets {
		l.evictLocked(now)
	}
	key := [2]string{ip, strings.ToLower(username)}
	if l.ipUser[key] == nil {
		l.ipUser[key] = &bucket{windowStart: now}
	}
	if l.ipOnly[ip] == nil {
		l.ipOnly[ip] = &bucket{windowStart: now}
	}
	userCount := l.ipUser[key].recordFailure(now)
	ipCount := l.ipOnly[ip].recordFailure(now)
	return userCount > ipUserLimit || ipCount > ipLimit
}

// Exceeded reports whether an attempt must be denied before credentials are checked.
func (l *LoginLimiter) Exceeded(ip, username string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	var userCount, ipCount uint32
	if b := l.ipUser[[2]string{ip, strings.ToLower(username)}]; b != nil {
		userCount = b.current(now)
	}
	if b := l.ipOnly[ip]; b != nil {
		ipCount = b.current(now)
	}
	return userCount >= ipUserLimit || ipCount >= ipLimit
}

// EvictExpired removes expired buckets to bound memory growth.
func (l *LoginLimiter) EvictExpired() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.evictLocked(l.now())
}

func (l *LoginLimiter) evictLocked(now int64) {
	for k, b := range l.ipUser {
		if now-b.windowStart >= limitWindow {
			delete(l.ipUser, k)
		}
	}
	for k, b := range l.ipOnly {
		if now-b.windowStart >= limitWindow {
			delete(l.ipOnly, k)
		}
	}
}

// ActiveBuckets is for observability only.
func (l *LoginLimiter) ActiveBuckets() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.ipUser) + len(l.ipOnly)
}
