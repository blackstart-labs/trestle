package identity

import (
	"fmt"
	"testing"
)

func ip(a int) string { return fmt.Sprintf("10.0.0.%d", a) }

func TestFreshLimiterAllowsFirstRequest(t *testing.T) {
	if NewLoginLimiter().Exceeded(ip(1), "alice") {
		t.Error("fresh limiter denies")
	}
}

func TestIPUserLimitTriggersAfterThreshold(t *testing.T) {
	l := NewLoginLimiter()
	for range ipUserLimit {
		l.RecordFailure(ip(1), "alice")
	}
	if !l.Exceeded(ip(1), "alice") {
		t.Error("limit not exceeded")
	}
}

func TestDifferentUserOnSameIPHasIndependentUserBucket(t *testing.T) {
	l := NewLoginLimiter()
	for range ipUserLimit {
		l.RecordFailure(ip(1), "alice")
	}
	// The IP-only bucket is still under ipLimit.
	if l.Exceeded(ip(1), "bob") {
		t.Error("bob is limited")
	}
}

func TestIPWideLimitTriggersAcrossUsernames(t *testing.T) {
	l := NewLoginLimiter()
	for i := range ipLimit {
		l.RecordFailure(ip(2), fmt.Sprintf("user%d", i))
	}
	if !l.Exceeded(ip(2), "newuser") {
		t.Error("IP-wide limit not exceeded")
	}
}

func TestDifferentIPsDoNotShareBuckets(t *testing.T) {
	l := NewLoginLimiter()
	for range ipUserLimit {
		l.RecordFailure(ip(1), "alice")
	}
	if l.Exceeded(ip(2), "alice") {
		t.Error("other IP is limited")
	}
}

func TestEvictExpiredClearsBuckets(t *testing.T) {
	l := NewLoginLimiter()
	now := int64(1_000_000)
	l.now = func() int64 { return now }
	l.RecordFailure(ip(1), "alice")
	if l.ActiveBuckets() == 0 {
		t.Fatal("no buckets recorded")
	}
	// Recent buckets survive eviction.
	l.EvictExpired()
	if l.ActiveBuckets() == 0 {
		t.Fatal("recent buckets evicted")
	}
	// Buckets older than the window are removed.
	now += limitWindow
	l.EvictExpired()
	if n := l.ActiveBuckets(); n != 0 {
		t.Errorf("%d buckets survive expiry", n)
	}
}

func TestUsernameIsCanonicalizedForBucketKey(t *testing.T) {
	l := NewLoginLimiter()
	for range ipUserLimit {
		l.RecordFailure(ip(1), "Alice")
	}
	if !l.Exceeded(ip(1), "alice") {
		t.Error("case variants use separate buckets")
	}
}

func TestRecordFailureEvictsExpiredBucketsPastTheCap(t *testing.T) {
	l := NewLoginLimiter()
	clock := int64(1_000_000)
	l.now = func() int64 { return clock }
	for i := 0; i <= maxBuckets; i++ {
		l.RecordFailure("10.0.0.1", fmt.Sprintf("user%d", i))
	}
	clock += limitWindow
	l.RecordFailure("10.0.0.2", "alice")
	if got := l.ActiveBuckets(); got != 2 {
		t.Fatalf("expected only the fresh buckets to remain, got %d", got)
	}
}
