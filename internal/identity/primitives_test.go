package identity

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

// Usernames (ported from src/identity/usernames.rs).

func mustUsername(t *testing.T, raw string) string {
	t.Helper()
	u, err := ParseUsername(raw)
	if err != nil {
		t.Fatalf("ParseUsername(%q): %v", raw, err)
	}
	return u
}

func wantUsernameErr(t *testing.T, raw string, want error) {
	t.Helper()
	if _, err := ParseUsername(raw); !errors.Is(err, want) {
		t.Errorf("ParseUsername(%q) = %v, want %v", raw, err, want)
	}
}

func TestLowercaseASCIIUsernameIsAccepted(t *testing.T) {
	if u := mustUsername(t, "alice"); u != "alice" {
		t.Errorf("got %q", u)
	}
}

func TestUppercaseIsNormalizedToLowercase(t *testing.T) {
	if u := mustUsername(t, "Alice"); u != "alice" {
		t.Errorf("got %q", u)
	}
}

func TestMixedCaseIsNormalized(t *testing.T) {
	if u := mustUsername(t, "ADMIN"); u != "admin" {
		t.Errorf("got %q", u)
	}
}

func TestDigitsDotsUnderscoresHyphensAccepted(t *testing.T) {
	if u := mustUsername(t, "user.name_01-ok"); u != "user.name_01-ok" {
		t.Errorf("got %q", u)
	}
}

func TestUsernameTooShortRejected(t *testing.T) {
	wantUsernameErr(t, "ab", ErrUsernameTooShort)
	wantUsernameErr(t, "", ErrUsernameTooShort)
}

func TestUsernameMinimumLengthAccepted(t *testing.T) { mustUsername(t, "abc") }

func TestUsernameTooLongRejected(t *testing.T) {
	wantUsernameErr(t, strings.Repeat("a", MaxUsernameLen+1), ErrUsernameTooLong)
}

func TestUsernameMaximumLengthAccepted(t *testing.T) {
	mustUsername(t, strings.Repeat("a", MaxUsernameLen))
}

func TestUsernameSpacesRejected(t *testing.T) { wantUsernameErr(t, "user name", ErrUsernameInvalid) }

func TestUsernameAtSignRejected(t *testing.T) { wantUsernameErr(t, "user@example", ErrUsernameInvalid) }

func TestUsernameNonASCIIRejected(t *testing.T) { wantUsernameErr(t, "ûser", ErrUsernameInvalid) }

func TestUppercaseAndLowercaseNormalizeToSame(t *testing.T) {
	if mustUsername(t, "testuser") != mustUsername(t, "TESTUSER") {
		t.Error("case variants differ")
	}
}

// Tokens (ported from src/identity/tokens.rs).

func mustToken(t *testing.T) (string, string) {
	t.Helper()
	token, hash, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	return token, hash
}

func TestTokenGeneratesAndHashes(t *testing.T) {
	token, hash := mustToken(t)
	// base64url without padding: 32 bytes -> 43 chars.
	if len(token) != 43 {
		t.Errorf("unexpected token length: %d", len(token))
	}
	if len(hash) != 64 {
		t.Errorf("unexpected hash length: %d", len(hash))
	}
}

func TestTwoGeneratedTokensAreDistinct(t *testing.T) {
	t1, h1 := mustToken(t)
	t2, h2 := mustToken(t)
	if t1 == t2 || h1 == h2 {
		t.Error("tokens or hashes collide")
	}
}

func TestHashOfSameTokenIsDeterministic(t *testing.T) {
	token, _ := mustToken(t)
	a, err1 := HashClientToken(token)
	b, err2 := HashClientToken(token)
	if err1 != nil || err2 != nil || a != b {
		t.Errorf("%q %v / %q %v", a, err1, b, err2)
	}
}

func TestClientHashMatchesServerHash(t *testing.T) {
	token, serverHash := mustToken(t)
	clientHash, err := HashClientToken(token)
	if err != nil || clientHash != serverHash {
		t.Errorf("client %q (%v), server %q", clientHash, err, serverHash)
	}
}

func TestInvalidBase64RejectedByClientHash(t *testing.T) {
	if _, err := HashClientToken("not-valid-base64!!!!"); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("got %v", err)
	}
}

func TestWrongLengthTokenRejected(t *testing.T) {
	short := base64.RawURLEncoding.EncodeToString([]byte("tooshort"))
	if _, err := HashClientToken(short); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("got %v", err)
	}
}

func TestTokenHashHasCorrectHexLength(t *testing.T) {
	_, hash := mustToken(t)
	if len(hash) != 64 {
		t.Errorf("length %d", len(hash))
	}
	if strings.Trim(hash, "0123456789abcdefABCDEF") != "" {
		t.Errorf("hash must be hex: %s", hash)
	}
}

// Constant-time comparison (ported from src/http/middleware.rs; cookie
// extraction is now net/http's Request.Cookie).

func TestConstantTimeEqMatchesEqualStrings(t *testing.T) {
	if !ConstantTimeEqual("abc123", "abc123") {
		t.Error("equal strings differ")
	}
}

func TestConstantTimeEqRejectsDifferentStrings(t *testing.T) {
	if ConstantTimeEqual("abc123", "abc124") {
		t.Error("different strings equal")
	}
}

func TestConstantTimeEqRejectsDifferentLengths(t *testing.T) {
	if ConstantTimeEqual("abc", "abcd") {
		t.Error("different lengths equal")
	}
}

func TestConstantTimeEqRejectsEmptyVsNonempty(t *testing.T) {
	if ConstantTimeEqual("", "a") || ConstantTimeEqual("a", "") {
		t.Error("empty equals nonempty")
	}
}

func TestConstantTimeEqAcceptsEmptyEmpty(t *testing.T) {
	if !ConstantTimeEqual("", "") {
		t.Error("empty strings differ")
	}
}

// Passwords (ported from src/identity/passwords.rs).

func testHasher(t *testing.T) Hasher {
	t.Helper()
	// Minimal parameters keep the tests fast.
	h, err := NewHasher(4096, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func mustHash(t *testing.T, h Hasher, password string) string {
	t.Helper()
	hash, err := h.Hash([]byte(password))
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func TestHashAndVerifyCorrectPassword(t *testing.T) {
	h := testHasher(t)
	hash := mustHash(t, h, "correct-password")
	if ok, err := h.Verify([]byte("correct-password"), hash); err != nil || !ok {
		t.Errorf("verify = %v, %v", ok, err)
	}
}

func TestVerifyWrongPasswordReturnsFalse(t *testing.T) {
	h := testHasher(t)
	hash := mustHash(t, h, "correct-password")
	if ok, err := h.Verify([]byte("wrong-password"), hash); err != nil || ok {
		t.Errorf("verify = %v, %v", ok, err)
	}
}

func TestHashRejectsTooShort(t *testing.T) {
	if _, err := testHasher(t).Hash([]byte("short")); !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("got %v", err)
	}
}

func TestHashRejectsTooLong(t *testing.T) {
	if _, err := testHasher(t).Hash(bytes.Repeat([]byte("a"), MaxPasswordBytes+1)); !errors.Is(err, ErrPasswordTooLong) {
		t.Errorf("got %v", err)
	}
}

func TestVerifyTooLongReturnsError(t *testing.T) {
	h := testHasher(t)
	hash := mustHash(t, h, "correct-password")
	if _, err := h.Verify(bytes.Repeat([]byte("a"), MaxPasswordBytes+1), hash); !errors.Is(err, ErrPasswordTooLong) {
		t.Errorf("got %v", err)
	}
}

func TestHashProducesArgon2idFormat(t *testing.T) {
	if hash := mustHash(t, testHasher(t), "test-password-ok"); !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("expected argon2id prefix, got: %s", hash)
	}
}

func TestTwoHashesOfSamePasswordDiffer(t *testing.T) {
	h := testHasher(t)
	if mustHash(t, h, "same-password-xx") == mustHash(t, h, "same-password-xx") {
		t.Error("different salts must produce different hashes")
	}
}

func TestMinimumLengthPasswordAccepted(t *testing.T) {
	mustHash(t, testHasher(t), strings.Repeat("a", MinPasswordBytes))
}

func TestBelowMinimumLengthRejected(t *testing.T) {
	if _, err := testHasher(t).Hash(bytes.Repeat([]byte("a"), MinPasswordBytes-1)); !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("got %v", err)
	}
}
