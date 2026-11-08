// Package identity owns users, sessions, CSRF tokens, passwords and login limits.
// HTTP handlers and CLI commands call into it; they never bypass it for identity data.
package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Usernames: 3–64 ASCII letters, digits, '.', '_', '-', '@', '+'; canonical form is lowercase.
// ---------------------------------------------------------------------------

const (
	MinUsernameLen = 3
	MaxUsernameLen = 64
)

var (
	ErrUsernameTooShort = fmt.Errorf("username must be at least %d characters", MinUsernameLen)
	ErrUsernameTooLong  = fmt.Errorf("username must be at most %d characters", MaxUsernameLen)
	ErrUsernameInvalid  = errors.New("username may only contain letters, digits, '.', '_', '-', '@', and '+'")
)

// ParseUsername validates and normalizes a raw username to lowercase.
func ParseUsername(raw string) (string, error) {
	if len(raw) < MinUsernameLen {
		return "", ErrUsernameTooShort
	}
	if len(raw) > MaxUsernameLen {
		return "", ErrUsernameTooLong
	}
	for i := 0; i < len(raw); i++ {
		b := raw[i]
		if !(b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '.' || b == '_' || b == '-' || b == '@' || b == '+') {
			return "", ErrUsernameInvalid
		}
	}
	return strings.ToLower(raw), nil
}

// ---------------------------------------------------------------------------
// Tokens: 32 random bytes, base64url without padding. Only the SHA-256 hex
// digest of the raw bytes is stored; the raw token is delivered once by cookie.
// ---------------------------------------------------------------------------

const TokenBytes = 32

var ErrInvalidToken = errors.New("invalid token format")

// GenerateToken returns a new token and its storable hash.
func GenerateToken() (token, hash string, err error) {
	b := make([]byte, TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", errors.New("token generation failed")
	}
	sum := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(b), hex.EncodeToString(sum[:]), nil
}

// HashClientToken hashes a token received from a client for lookup or comparison.
func HashClientToken(token string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(b) != TokenBytes {
		return "", ErrInvalidToken
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// ConstantTimeEqual compares two strings without leaking timing.
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ---------------------------------------------------------------------------
// Passwords: Argon2id, PHC string format ($argon2id$v=19$m=..,t=..,p=..$salt$hash).
// ---------------------------------------------------------------------------

const (
	MinPasswordBytes = 8
	MaxPasswordBytes = 512
)

var (
	ErrPasswordTooShort = errors.New("password too short")
	ErrPasswordTooLong  = errors.New("password too long")
	ErrPasswordInternal = errors.New("password processing error")
)

type Hasher struct {
	M, T uint32
	P    uint8
}

// NewHasher validates the parameters against Argon2's accepted ranges.
func NewHasher(m, t, p uint32) (Hasher, error) {
	if p < 1 || p > 255 || t < 1 || m < 8*p {
		return Hasher{}, errors.New("argon2 parameter error")
	}
	return Hasher{M: m, T: t, P: uint8(p)}, nil
}

func DefaultHasher() Hasher { return Hasher{M: 65_536, T: 2, P: 1} }

func (h Hasher) Hash(password []byte) (string, error) {
	if len(password) < MinPasswordBytes {
		return "", ErrPasswordTooShort
	}
	if len(password) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", ErrPasswordInternal
	}
	key := argon2.IDKey(password, salt, h.T, h.M, h.P, 32)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", h.M, h.T, h.P,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// Verify checks a password against a stored hash, using the parameters in the hash.
func (h Hasher) Verify(password []byte, stored string) (bool, error) {
	if len(password) > MaxPasswordBytes {
		return false, ErrPasswordTooLong
	}
	// Support bcrypt for imported hashes starting with $2a$, $2b$, or $2y$ (e.g. from Leantime migration)
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
		normalized := stored
		if strings.HasPrefix(stored, "$2y$") {
			normalized = "$2a$" + stored[4:]
		}
		err := bcrypt.CompareHashAndPassword([]byte(normalized), password)
		if err == nil {
			return true, nil
		}
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	parts := strings.Split(stored, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, ErrPasswordInternal
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil || p == 0 || t == 0 {
		return false, ErrPasswordInternal
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[4])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[5])
	if err1 != nil || err2 != nil || len(want) == 0 {
		return false, ErrPasswordInternal
	}
	got := argon2.IDKey(password, salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// DummyVerify equalises timing when no user record exists.
func (h Hasher) DummyVerify() {
	const dummy = "$argon2id$v=19$m=65536,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	_, _ = h.Verify([]byte("dummy-check-password"), dummy)
}
