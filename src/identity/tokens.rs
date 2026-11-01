/// Session and CSRF token generation, hashing, and validation.
///
/// Session tokens are 32 cryptographically random bytes, base64url-encoded
/// (without padding) for transmission. Only the SHA-256 hex digest of the raw
/// token bytes is stored in the database. The raw token is delivered once via
/// HttpOnly cookie and never persisted again.
///
/// CSRF tokens follow the same format but are delivered via a non-HttpOnly
/// cookie so that JavaScript can read and forward them in the X-CSRF-Token header.
/// The server stores the SHA-256 hex digest in the session row for validation.
use base64ct::{Base64UrlUnpadded, Encoding};
use rand::TryRngCore;
use sha2::{Digest, Sha256};

/// Length of the raw random token in bytes.
pub const TOKEN_BYTES: usize = 32;
/// Length of the hex-encoded SHA-256 digest, in characters (32 bytes → 64 hex chars).
pub const HASH_HEX_LEN: usize = 64;

/// An opaque token string (base64url, 43 characters for 32 bytes).
#[derive(Clone)]
pub struct Token(String);

impl Token {
    /// Generate a new cryptographically random token.
    ///
    /// # Errors
    /// Returns an error if the OS RNG is unavailable.
    pub fn generate() -> Result<Self, TokenError> {
        let mut bytes = [0u8; TOKEN_BYTES];
        rand::rngs::OsRng
            .try_fill_bytes(&mut bytes)
            .map_err(|_| TokenError::RngFailure)?;
        // base64url without padding for URL/cookie safety.
        Ok(Self(Base64UrlUnpadded::encode_string(&bytes)))
    }

    /// Return the base64url string for transmission (cookie value, HTTP response).
    ///
    /// Never log or persist this value.
    pub fn as_str(&self) -> &str {
        &self.0
    }

    /// Compute the SHA-256 hex digest of the raw bytes for database storage.
    ///
    /// The digest is what gets stored, not the token itself.
    pub fn hash(&self) -> TokenHash {
        // Decode back to bytes for hashing.
        // Unwrap: we always encode valid base64url, so decode must succeed.
        let bytes =
            Base64UrlUnpadded::decode_vec(&self.0).expect("self-generated token must decode");
        let digest = Sha256::digest(&bytes);
        let hex = hex_encode(&digest);
        TokenHash(hex)
    }
}

/// Compute the hash of a token string received from a client.
///
/// This is used to look up the session row without storing the raw token.
///
/// # Errors
/// Returns `TokenError::InvalidFormat` if the token string is not valid base64url
/// or does not decode to exactly `TOKEN_BYTES` bytes.
pub fn hash_client_token(token_str: &str) -> Result<TokenHash, TokenError> {
    let bytes = Base64UrlUnpadded::decode_vec(token_str).map_err(|_| TokenError::InvalidFormat)?;
    if bytes.len() != TOKEN_BYTES {
        return Err(TokenError::InvalidFormat);
    }
    let digest = Sha256::digest(&bytes);
    Ok(TokenHash(hex_encode(&digest)))
}

/// The SHA-256 hex digest of a token, safe to store and log.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TokenHash(String);

impl TokenHash {
    /// Return the hex-encoded digest string for database storage.
    pub fn as_str(&self) -> &str {
        &self.0
    }
}

impl std::fmt::Display for TokenHash {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(&self.0)
    }
}

#[derive(Debug, PartialEq, Eq)]
pub enum TokenError {
    /// OS RNG is unavailable.
    RngFailure,
    /// Token string is not valid base64url or wrong length.
    InvalidFormat,
}

impl std::fmt::Display for TokenError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::RngFailure => f.write_str("token generation failed"),
            Self::InvalidFormat => f.write_str("invalid token format"),
        }
    }
}

impl std::error::Error for TokenError {}

fn hex_encode(bytes: &[u8]) -> String {
    bytes
        .iter()
        .fold(String::with_capacity(bytes.len() * 2), |mut acc, b| {
            use std::fmt::Write;
            write!(acc, "{b:02x}").unwrap();
            acc
        })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn token_generates_and_hashes() {
        let token = Token::generate().unwrap();
        let s = token.as_str();
        // base64url without padding: 32 bytes → ceil(32*4/3) = 43 chars (no padding).
        assert_eq!(s.len(), 43, "unexpected token length: {}", s.len());
        let hash = token.hash();
        assert_eq!(hash.as_str().len(), HASH_HEX_LEN);
    }

    #[test]
    fn two_generated_tokens_are_distinct() {
        let t1 = Token::generate().unwrap();
        let t2 = Token::generate().unwrap();
        assert_ne!(t1.as_str(), t2.as_str());
        assert_ne!(t1.hash(), t2.hash());
    }

    #[test]
    fn hash_of_same_token_is_deterministic() {
        let token = Token::generate().unwrap();
        assert_eq!(token.hash(), token.hash());
    }

    #[test]
    fn client_hash_matches_server_hash() {
        let token = Token::generate().unwrap();
        let server_hash = token.hash();
        let client_hash = hash_client_token(token.as_str()).unwrap();
        assert_eq!(server_hash, client_hash);
    }

    #[test]
    fn invalid_base64_rejected_by_client_hash() {
        assert_eq!(
            hash_client_token("not-valid-base64!!!!").unwrap_err(),
            TokenError::InvalidFormat
        );
    }

    #[test]
    fn wrong_length_token_rejected() {
        // Valid base64url but wrong decoded length.
        let short = Base64UrlUnpadded::encode_string(b"tooshort");
        assert_eq!(
            hash_client_token(&short).unwrap_err(),
            TokenError::InvalidFormat
        );
    }

    #[test]
    fn token_hash_has_correct_hex_length() {
        let token = Token::generate().unwrap();
        let hash = token.hash();
        assert_eq!(hash.as_str().len(), 64);
        assert!(
            hash.as_str().chars().all(|c| c.is_ascii_hexdigit()),
            "hash must be hex: {}",
            hash.as_str()
        );
    }
}
