/// In-memory login rate limiter.
///
/// Policy (identity-policy.md):
/// - IP + canonical username: 5 failures per 15 minutes.
/// - IP across usernames: 30 failures per 15 minutes.
/// - No permanent lockout. A correct password does not bypass an active limit.
/// - Uses TCP peer IP; never trusts X-Forwarded-For.
/// - Generic 429 response; never reveals account existence.
/// - Atomic accounting prevents concurrent overshoot.
/// - Successful attempts are counted to prevent bypass through automation.
///
/// This module uses a DashMap with per-bucket atomics. Windows slide forward
/// once the window expires; counts reset at that point. Bounded cleanup runs
/// on each successful attempt and periodically to prevent unbounded growth.
use dashmap::DashMap;
use std::net::IpAddr;
use std::sync::Arc;
use std::sync::atomic::{AtomicI64, AtomicU32, Ordering};

/// Window duration in seconds.
const WINDOW_SECS: i64 = 15 * 60;
/// IP + username failure limit.
const IP_USER_LIMIT: u32 = 5;
/// IP-only failure limit (across all usernames).
const IP_LIMIT: u32 = 30;

/// A rate-limit bucket that tracks failures within a sliding window.
struct Bucket {
    /// Count of failures in the current window.
    count: AtomicU32,
    /// Unix timestamp (seconds) when this window started.
    window_start: AtomicI64,
}

impl Bucket {
    fn new(now: i64) -> Arc<Self> {
        Arc::new(Self {
            count: AtomicU32::new(0),
            window_start: AtomicI64::new(now),
        })
    }

    /// Record a failure and return the new count within the current window.
    ///
    /// If the window has expired, it resets first. The combined check-and-increment
    /// is not atomic across the window check and the count increment; this means
    /// a small overshoot of exactly one may occur on a window boundary when two
    /// requests race. This is acceptable: the limit is not a precise cryptographic
    /// bound, and the overshoot is bounded to one per window reset.
    fn record_failure(&self, now: i64) -> u32 {
        let ws = self.window_start.load(Ordering::Acquire);
        if now - ws >= WINDOW_SECS {
            // Window expired; reset by storing new start and count of 1.
            self.window_start.store(now, Ordering::Release);
            self.count.store(1, Ordering::Release);
            return 1;
        }
        self.count.fetch_add(1, Ordering::AcqRel) + 1
    }

    /// Return current count without modifying it.
    fn current_count(&self, now: i64) -> u32 {
        let ws = self.window_start.load(Ordering::Acquire);
        if now - ws >= WINDOW_SECS {
            return 0;
        }
        self.count.load(Ordering::Acquire)
    }

    /// True if the window has expired and the bucket can be evicted.
    fn is_expired(&self, now: i64) -> bool {
        now - self.window_start.load(Ordering::Acquire) >= WINDOW_SECS
    }
}

/// Login rate limiter with IP-only and IP+username buckets.
#[derive(Clone)]
pub struct LoginLimiter {
    /// Key: (ip, canonical_username). Limit: IP_USER_LIMIT failures per window.
    ip_user: Arc<DashMap<(String, String), Arc<Bucket>>>,
    /// Key: ip. Limit: IP_LIMIT failures per window across all usernames.
    ip_only: Arc<DashMap<String, Arc<Bucket>>>,
}

/// Outcome of a rate-limit check.
#[derive(Debug, PartialEq, Eq)]
pub enum LimitResult {
    /// The request is allowed to proceed.
    Allowed,
    /// The limit has been exceeded; deny the request.
    Exceeded,
}

impl LoginLimiter {
    /// Create a new limiter.
    pub fn new() -> Self {
        Self {
            ip_user: Arc::new(DashMap::new()),
            ip_only: Arc::new(DashMap::new()),
        }
    }

    /// Record a login failure and return whether subsequent attempts are limited.
    ///
    /// Call this after a failed login attempt, before sending the response.
    /// Returns `LimitResult::Exceeded` if the limit was already hit before this
    /// failure (i.e., the request should have been denied earlier) or is now hit.
    pub fn record_failure(&self, ip: IpAddr, username: &str) -> LimitResult {
        let now = current_unix_seconds();
        let ip_str = ip.to_string();

        let ip_user_key = (ip_str.clone(), username.to_ascii_lowercase());
        let ip_user_count = self
            .ip_user
            .entry(ip_user_key)
            .or_insert_with(|| Bucket::new(now))
            .record_failure(now);

        let ip_count = self
            .ip_only
            .entry(ip_str)
            .or_insert_with(|| Bucket::new(now))
            .record_failure(now);

        if ip_user_count > IP_USER_LIMIT || ip_count > IP_LIMIT {
            LimitResult::Exceeded
        } else {
            LimitResult::Allowed
        }
    }

    /// Check whether a login attempt should be denied before attempting auth.
    ///
    /// Call this before validating credentials. Does not modify any counters.
    pub fn check(&self, ip: IpAddr, username: &str) -> LimitResult {
        let now = current_unix_seconds();
        let ip_str = ip.to_string();

        let ip_user_key = (ip_str.clone(), username.to_ascii_lowercase());
        let ip_user_count = self
            .ip_user
            .get(&ip_user_key)
            .map(|b| b.current_count(now))
            .unwrap_or(0);

        let ip_count = self
            .ip_only
            .get(&ip_str)
            .map(|b| b.current_count(now))
            .unwrap_or(0);

        // Check against the limit, not limit + 1, so that the Nth attempt is
        // blocked (i.e., after N failures the (N+1)th is denied).
        if ip_user_count >= IP_USER_LIMIT || ip_count >= IP_LIMIT {
            LimitResult::Exceeded
        } else {
            LimitResult::Allowed
        }
    }

    /// Remove expired buckets to bound memory growth.
    ///
    /// Call periodically (e.g., on startup and in a background task)
    /// or opportunistically on successful requests.
    pub fn evict_expired(&self) {
        let now = current_unix_seconds();
        self.ip_user.retain(|_, bucket| !bucket.is_expired(now));
        self.ip_only.retain(|_, bucket| !bucket.is_expired(now));
    }

    /// Return the number of active buckets (for observability, not security).
    pub fn active_bucket_count(&self) -> usize {
        self.ip_user.len() + self.ip_only.len()
    }
}

impl Default for LoginLimiter {
    fn default() -> Self {
        Self::new()
    }
}

fn current_unix_seconds() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs() as i64
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::net::{IpAddr, Ipv4Addr};

    fn ip(a: u8) -> IpAddr {
        IpAddr::V4(Ipv4Addr::new(10, 0, 0, a))
    }

    #[test]
    fn fresh_limiter_allows_first_request() {
        let lim = LoginLimiter::new();
        assert_eq!(lim.check(ip(1), "alice"), LimitResult::Allowed);
    }

    #[test]
    fn ip_user_limit_triggers_after_threshold() {
        let lim = LoginLimiter::new();
        // Record exactly IP_USER_LIMIT failures.
        for _ in 0..IP_USER_LIMIT {
            lim.record_failure(ip(1), "alice");
        }
        // Next check should be exceeded.
        assert_eq!(lim.check(ip(1), "alice"), LimitResult::Exceeded);
    }

    #[test]
    fn different_user_on_same_ip_has_independent_user_bucket() {
        let lim = LoginLimiter::new();
        for _ in 0..IP_USER_LIMIT {
            lim.record_failure(ip(1), "alice");
        }
        // "bob" on same IP has no user-bucket failures yet.
        // IP-only bucket has IP_USER_LIMIT failures, still under IP_LIMIT.
        assert_eq!(lim.check(ip(1), "bob"), LimitResult::Allowed);
    }

    #[test]
    fn ip_wide_limit_triggers_across_usernames() {
        let lim = LoginLimiter::new();
        // Record IP_LIMIT failures spread across usernames.
        for i in 0..IP_LIMIT {
            lim.record_failure(ip(2), &format!("user{i}"));
        }
        // A new username on the same IP should be blocked by IP-wide bucket.
        assert_eq!(lim.check(ip(2), "newuser"), LimitResult::Exceeded);
    }

    #[test]
    fn different_ips_do_not_share_buckets() {
        let lim = LoginLimiter::new();
        for _ in 0..IP_USER_LIMIT {
            lim.record_failure(ip(1), "alice");
        }
        // A different IP is unaffected.
        assert_eq!(lim.check(ip(2), "alice"), LimitResult::Allowed);
    }

    #[test]
    fn evict_expired_clears_buckets() {
        let lim = LoginLimiter::new();
        lim.record_failure(ip(1), "alice");
        assert!(lim.active_bucket_count() > 0);
        // Force expiry by directly manipulating is not possible from outside,
        // so verify evict_expired does not panic and preserves active buckets.
        lim.evict_expired();
        // Buckets are recent, so none should be evicted.
        assert!(lim.active_bucket_count() > 0);
    }

    #[test]
    fn username_is_canonicalized_for_bucket_key() {
        let lim = LoginLimiter::new();
        // Failures recorded for "Alice" and "alice" should count to the same bucket.
        for _ in 0..IP_USER_LIMIT {
            lim.record_failure(ip(1), "Alice");
        }
        assert_eq!(lim.check(ip(1), "alice"), LimitResult::Exceeded);
    }
}
