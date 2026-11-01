/// Argon2id password hashing with configurable parameters.
///
/// Parameters are chosen at startup from configuration and measured on this
/// host before sign-off. Raw passwords are never stored, logged, or returned.
/// The policy requires unbounded trimming and normalization to be absent;
/// passwords are hashed as-received (after length validation).
use argon2::{Algorithm, Argon2, Params, PasswordHasher, Version};
use rand::TryRngCore;

/// Checked: OWASP minimum is m=19456, t=2, p=1.
/// Host measurement (2026-11-01): m=65536,t=1 = 71ms; m=65536,t=2 = 122ms avg.
/// Default matches the approved policy measurement: 64 MiB memory, 2 passes.
pub const DEFAULT_M_COST: u32 = 65_536;
pub const DEFAULT_T_COST: u32 = 2;
pub const DEFAULT_PARALLELISM: u32 = 1;

/// Minimum/maximum password lengths enforced before hashing.
/// The minimum prevents trivially weak passwords; the maximum prevents DoS
/// by limiting Argon2id work on attacker-controlled inputs.
pub const MIN_PASSWORD_BYTES: usize = 8;
pub const MAX_PASSWORD_BYTES: usize = 512;

/// Argon2id hasher bound to specific parameters for this instance.
#[derive(Clone)]
pub struct Hasher {
    params: Params,
}

/// Public error type; never exposes internal details.
#[derive(Debug, PartialEq, Eq)]
pub enum HashError {
    /// Password is shorter than the minimum allowed length.
    TooShort,
    /// Password exceeds the maximum allowed length.
    TooLong,
    /// Internal hashing failure; logged separately with context.
    Internal,
}

impl std::fmt::Display for HashError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::TooShort => f.write_str("password too short"),
            Self::TooLong => f.write_str("password too long"),
            Self::Internal => f.write_str("password processing error"),
        }
    }
}

impl std::error::Error for HashError {}

impl Hasher {
    /// Create a hasher with explicit parameters.
    ///
    /// # Errors
    /// Returns an error if the parameters are out of Argon2id's allowed ranges.
    pub fn new(m_cost: u32, t_cost: u32, parallelism: u32) -> Result<Self, String> {
        let params = Params::new(m_cost, t_cost, parallelism, None)
            .map_err(|e| format!("argon2 parameter error: {e}"))?;
        Ok(Self { params })
    }

    /// Create a hasher with the default approved parameters.
    pub fn default_params() -> Self {
        // Unwrap is safe: constants are within Argon2id's accepted ranges.
        Self::new(DEFAULT_M_COST, DEFAULT_T_COST, DEFAULT_PARALLELISM).unwrap()
    }

    /// Hash a password with a fresh random 16-byte salt.
    ///
    /// Returns the encoded hash string in `$argon2id$...` format.
    ///
    /// # Errors
    /// Returns `HashError::TooShort` or `HashError::TooLong` for invalid lengths,
    /// `HashError::Internal` on RNG or hashing failures.
    pub fn hash(&self, password: &[u8]) -> Result<String, HashError> {
        if password.len() < MIN_PASSWORD_BYTES {
            return Err(HashError::TooShort);
        }
        if password.len() > MAX_PASSWORD_BYTES {
            return Err(HashError::TooLong);
        }
        let mut salt_bytes = [0u8; 16];
        rand::rngs::OsRng
            .try_fill_bytes(&mut salt_bytes)
            .map_err(|_| HashError::Internal)?;
        let salt = argon2::password_hash::SaltString::encode_b64(&salt_bytes)
            .map_err(|_| HashError::Internal)?;
        let argon2 = Argon2::new(Algorithm::Argon2id, Version::V0x13, self.params.clone());
        argon2
            .hash_password(password, &salt)
            .map(|h| h.to_string())
            .map_err(|_| HashError::Internal)
    }

    /// Verify a password against a stored hash.
    ///
    /// Returns `Ok(true)` if the password matches, `Ok(false)` if it does not.
    /// Always performs the full verification computation to avoid timing leaks.
    ///
    /// # Errors
    /// Returns `HashError::TooLong` to bound attacker input; shorter than minimum
    /// returns `Ok(false)` to avoid disclosing account existence through error type.
    pub fn verify(&self, password: &[u8], stored_hash: &str) -> Result<bool, HashError> {
        if password.len() > MAX_PASSWORD_BYTES {
            return Err(HashError::TooLong);
        }
        let parsed = argon2::PasswordHash::new(stored_hash).map_err(|_| HashError::Internal)?;
        // argon2::PasswordHasher::verify_password performs constant-time comparison.
        let argon2 = Argon2::new(Algorithm::Argon2id, Version::V0x13, self.params.clone());
        match argon2::PasswordVerifier::verify_password(&argon2, password, &parsed) {
            Ok(()) => Ok(true),
            Err(argon2::password_hash::Error::Password) => Ok(false),
            Err(_) => Err(HashError::Internal),
        }
    }

    /// Perform a dummy verification to equalise timing for unknown usernames.
    ///
    /// Callers MUST call this when no user record is found, to prevent timing
    /// side-channel distinguishing between unknown users and wrong passwords.
    pub fn dummy_verify(&self) {
        // Use a realistic-looking hash with correct parameters but a constant
        // that no real password will ever match.
        static DUMMY_HASH: &str = "$argon2id$v=19$m=65536,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA";
        let _ = self.verify(b"dummy-check-password", DUMMY_HASH);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn hasher() -> Hasher {
        // Use minimal parameters in tests to keep them fast.
        Hasher::new(4096, 1, 1).unwrap()
    }

    #[test]
    fn hash_and_verify_correct_password() {
        let h = hasher();
        let hash = h.hash(b"correct-password").unwrap();
        assert!(h.verify(b"correct-password", &hash).unwrap());
    }

    #[test]
    fn verify_wrong_password_returns_false() {
        let h = hasher();
        let hash = h.hash(b"correct-password").unwrap();
        assert!(!h.verify(b"wrong-password", &hash).unwrap());
    }

    #[test]
    fn hash_rejects_too_short() {
        let h = hasher();
        assert_eq!(h.hash(b"short").unwrap_err(), HashError::TooShort);
    }

    #[test]
    fn hash_rejects_too_long() {
        let h = hasher();
        assert_eq!(
            h.hash(&[b'a'; MAX_PASSWORD_BYTES + 1]).unwrap_err(),
            HashError::TooLong
        );
    }

    #[test]
    fn verify_too_long_returns_error() {
        let h = hasher();
        let hash = h.hash(b"correct-password").unwrap();
        assert_eq!(
            h.verify(&[b'a'; MAX_PASSWORD_BYTES + 1], &hash)
                .unwrap_err(),
            HashError::TooLong
        );
    }

    #[test]
    fn hash_produces_argon2id_format() {
        let h = hasher();
        let hash = h.hash(b"test-password-ok").unwrap();
        assert!(
            hash.starts_with("$argon2id$"),
            "expected argon2id prefix, got: {hash}"
        );
    }

    #[test]
    fn two_hashes_of_same_password_differ() {
        let h = hasher();
        let hash1 = h.hash(b"same-password-xx").unwrap();
        let hash2 = h.hash(b"same-password-xx").unwrap();
        assert_ne!(
            hash1, hash2,
            "different salts must produce different hashes"
        );
    }

    #[test]
    fn minimum_length_password_accepted() {
        let h = hasher();
        // Exactly MIN_PASSWORD_BYTES should succeed.
        let pw = vec![b'a'; MIN_PASSWORD_BYTES];
        assert!(h.hash(&pw).is_ok());
    }

    #[test]
    fn below_minimum_length_rejected() {
        let h = hasher();
        let pw = vec![b'a'; MIN_PASSWORD_BYTES - 1];
        assert_eq!(h.hash(&pw).unwrap_err(), HashError::TooShort);
    }
}
