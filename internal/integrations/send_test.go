package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestValidateDestinations(t *testing.T) {
	for _, tc := range []struct {
		provider, secret, chat string
		valid                  bool
	}{
		{"telegram", "123456:abcdefghijklmnopqrstuvwxyz_123456789", "-100123456", true},
		{"telegram", "123456:abcdefghijklmnopqrstuvwxyz_123456789", "-100123456:42", true},
		{"telegram", "123456:abcdefghijklmnopqrstuvwxyz_123456789", "-100123456/42", true},
		{"telegram", "123456:abcdefghijklmnopqrstuvwxyz_123456789", "@channel_name", true},
		{"slack", "https://hooks.slack.com/services/TEAM/BOT/TOKEN", "", true},
		{"slack", "https://hooks.slack-gov.com/services/TEAM/BOT/TOKEN", "", true},
		{"discord", "https://discord.com/api/webhooks/123/token_A-1", "", true},
		{"discord", "https://discord.com/api/v10/webhooks/123/token_A-1", "", true},
		{"slack", "http://hooks.slack.com/services/T/B/T", "", false},
		{"slack", "https://hooks.slack.com.evil.test/services/T/B/T", "", false},
		{"slack", "https://hooks.slack.com:443/services/T/B/T", "", false},
		{"slack", "https://user@hooks.slack.com/services/T/B/T", "", false},
		{"slack", "https://hooks.slack.com/services/T/B/T?redirect=x", "", false},
		{"slack", "https://hooks.slack.com/services/T/B/T#fragment", "", false},
		{"discord", "https://127.0.0.1/api/webhooks/123/token", "", false},
		{"discord", "https://discord.com/api/webhooks/../token", "", false},
		{"discord", "https://discord.com/api/webhooks/123/token", "unexpected", false},
		{"telegram", "123:bad/token", "-123", false},
		{"telegram", "123456:abcdefghijklmnopqrstuvwxyz_123456789", "chat with spaces", false},
		{"unknown", "https://discord.com/api/webhooks/123/token", "", false},
	} {
		if got := Validate(tc.provider, tc.secret, tc.chat) == nil; got != tc.valid {
			t.Errorf("validation for %s: got %v want %v", tc.provider, got, tc.valid)
		}
	}
}

func TestProviderPayloadsAndFailures(t *testing.T) {
	for _, provider := range []string{"telegram", "slack", "discord"} {
		t.Run(provider, func(t *testing.T) {
			secret, chat := "https://hooks.slack.com/services/T/B/TOKEN", ""
			if provider == "telegram" {
				secret, chat = "123456:abcdefghijklmnopqrstuvwxyz_123456789", "-123"
			}
			if provider == "discord" {
				secret = "https://discord.com/api/webhooks/123/token"
			}
			for _, mode := range []string{"success", "rejected", "network", "bad_json", "api_error"} {
				called := false
				c := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
					called = true
					if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
						t.Fatal("invalid request")
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					switch provider {
					case "telegram":
						if body["chat_id"] != chat || body["text"] != "Task @everyone <https://example.com>" || !strings.HasSuffix(r.URL.Path, "/sendMessage") {
							t.Fatal(body)
						}
					case "slack":
						if body["mrkdwn"] != false || strings.Contains(body["text"].(string), "<") {
							t.Fatal(body)
						}
					case "discord":
						if len(body["allowed_mentions"].(map[string]any)["parse"].([]any)) != 0 || r.URL.Query().Get("wait") != "true" {
							t.Fatal(body)
						}
					}
					if mode == "network" {
						return nil, errors.New("secret-bearing URL: " + secret)
					}
					status, response := 200, `{"ok":true}`
					if mode == "rejected" {
						status, response = 429, secret
					}
					if mode == "bad_json" {
						response = "invalid"
					}
					if mode == "api_error" {
						response = `{"ok":false}`
					}
					return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(response)), Header: http.Header{}}, nil
				})}
				err := send(context.Background(), c, provider, secret, chat, "", "Task @everyone <https://example.com>")
				wantError := mode == "network" || mode == "rejected" || (provider == "telegram" && mode != "success")
				if !called || (err != nil) != wantError {
					t.Fatalf("%s: unexpected result %v", mode, err)
				}
				if err != nil && strings.Contains(err.Error(), secret) {
					t.Fatal("credential leaked")
				}
			}
		})
	}
}

func TestPrivateConnectionsAreBlocked(t *testing.T) {
	for _, address := range []string{"127.0.0.1:80", "[::1]:80", "10.0.0.1:80", "169.254.169.254:80"} {
		if conn, err := publicDial(context.Background(), "tcp", address); err == nil {
			conn.Close()
			t.Fatalf("private address allowed: %s", address)
		}
	}
}

func TestValidateWithTopic(t *testing.T) {
	const secret = "123456:abcdefghijklmnopqrstuvwxyz_123456789"
	if err := ValidateWithTopic("telegram", secret, "-100123456", "42"); err != nil {
		t.Fatalf("valid topic failed: %v", err)
	}
	if err := ValidateWithTopic("telegram", secret, "-100123456", "invalid_topic"); err == nil {
		t.Fatal("invalid topic was accepted")
	}
}

func TestTelegramTopicPayload(t *testing.T) {
	called := false
	c := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		called = true
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["chat_id"] != "-100123456" {
			t.Fatalf("got chat_id %v, want -100123456", body["chat_id"])
		}
		if threadID, ok := body["message_thread_id"].(float64); !ok || int64(threadID) != 42 {
			t.Fatalf("got message_thread_id %v, want 42", body["message_thread_id"])
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: http.Header{}}, nil
	})}
	err := send(context.Background(), c, "telegram", "123456:abcdefghijklmnopqrstuvwxyz_123456789", "-100123456", "42", "Test topic message")
	if err != nil || !called {
		t.Fatalf("send with topic failed: %v", err)
	}
}
