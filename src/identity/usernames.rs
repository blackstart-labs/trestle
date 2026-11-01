//! Username normalization and validation.
//!
//! Policy (from identity-policy.md):
//! - 3–64 ASCII characters: letters, digits, dot, underscore, hyphen.
//! - Accept uppercase input; canonicalize to ASCII lowercase.
//! - Uniqueness enforced on the canonical (lowercase) representation.
//! - Do not add leading-character restrictions or silently strip input.

/// Minimum and maximum length of a username, in bytes.
pub const MIN_USERNAME_LEN: usize = 3;
pub const MAX_USERNAME_LEN: usize = 64;

/// A validated, normalized (lowercase) username.
#[derive(Debug, Clone, PartialEq, Eq, Hash)]
pub struct Username(String);

impl Username {
    /// Normalize and validate a raw username string.
    ///
    /// Accepts mixed-case ASCII; normalizes to lowercase. Rejects empty strings,
    /// strings outside the allowed length range, and strings containing
    /// characters outside `[a-zA-Z0-9._-]`.
    ///
    /// # Errors
    /// Returns a `UsernameError` describing the first validation failure.
    pub fn parse(raw: &str) -> Result<Self, UsernameError> {
        if raw.len() < MIN_USERNAME_LEN {
            return Err(UsernameError::TooShort);
        }
        if raw.len() > MAX_USERNAME_LEN {
            return Err(UsernameError::TooLong);
        }
        if !raw
            .bytes()
            .all(|b| matches!(b, b'a'..=b'z' | b'A'..=b'Z' | b'0'..=b'9' | b'.' | b'_' | b'-'))
        {
            return Err(UsernameError::InvalidCharacter);
        }
        Ok(Self(raw.to_ascii_lowercase()))
    }

    /// Return the canonical (lowercase) string representation.
    pub fn as_str(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Display for Username {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}

/// Errors returned by [`Username::parse`].
#[derive(Debug, PartialEq, Eq, Clone, Copy)]
pub enum UsernameError {
    TooShort,
    TooLong,
    InvalidCharacter,
}

impl std::fmt::Display for UsernameError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::TooShort => write!(f, "username must be at least {MIN_USERNAME_LEN} characters"),
            Self::TooLong => write!(f, "username must be at most {MAX_USERNAME_LEN} characters"),
            Self::InvalidCharacter => {
                f.write_str("username may only contain letters, digits, '.', '_', and '-'")
            }
        }
    }
}

impl std::error::Error for UsernameError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn lowercase_ascii_username_is_accepted() {
        let u = Username::parse("alice").unwrap();
        assert_eq!(u.as_str(), "alice");
    }

    #[test]
    fn uppercase_is_normalized_to_lowercase() {
        let u = Username::parse("Alice").unwrap();
        assert_eq!(u.as_str(), "alice");
    }

    #[test]
    fn mixed_case_is_normalized() {
        let u = Username::parse("ADMIN").unwrap();
        assert_eq!(u.as_str(), "admin");
    }

    #[test]
    fn digits_dots_underscores_hyphens_accepted() {
        let u = Username::parse("user.name_01-ok").unwrap();
        assert_eq!(u.as_str(), "user.name_01-ok");
    }

    #[test]
    fn too_short_rejected() {
        assert_eq!(Username::parse("ab").unwrap_err(), UsernameError::TooShort);
        assert_eq!(Username::parse("").unwrap_err(), UsernameError::TooShort);
    }

    #[test]
    fn minimum_length_accepted() {
        assert!(Username::parse("abc").is_ok());
    }

    #[test]
    fn too_long_rejected() {
        let long = "a".repeat(MAX_USERNAME_LEN + 1);
        assert_eq!(Username::parse(&long).unwrap_err(), UsernameError::TooLong);
    }

    #[test]
    fn maximum_length_accepted() {
        let maxlen = "a".repeat(MAX_USERNAME_LEN);
        assert!(Username::parse(&maxlen).is_ok());
    }

    #[test]
    fn spaces_rejected() {
        assert_eq!(
            Username::parse("user name").unwrap_err(),
            UsernameError::InvalidCharacter
        );
    }

    #[test]
    fn at_sign_rejected() {
        assert_eq!(
            Username::parse("user@example").unwrap_err(),
            UsernameError::InvalidCharacter
        );
    }

    #[test]
    fn non_ascii_rejected() {
        assert_eq!(
            Username::parse("ûser").unwrap_err(),
            UsernameError::InvalidCharacter
        );
    }

    #[test]
    fn uppercase_and_lowercase_normalize_to_same() {
        let lower = Username::parse("testuser").unwrap();
        let upper = Username::parse("TESTUSER").unwrap();
        assert_eq!(lower, upper);
    }
}
