// Package integrations delivers plain-text project notifications to fixed providers.
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	botToken    = regexp.MustCompile(`^[0-9]{1,20}:[A-Za-z0-9_-]{20,100}$`)
	chatID      = regexp.MustCompile(`^(-?[0-9]{1,20}|@[A-Za-z0-9_]{5,32})$`)
	slackPath   = regexp.MustCompile(`^/services/[A-Za-z0-9]+/[A-Za-z0-9]+/[A-Za-z0-9]+$`)
	discordPath = regexp.MustCompile(`^/api(?:/v[0-9]+)?/webhooks/[0-9]+/[A-Za-z0-9_-]+$`)
)

func ValidProvider(provider string) bool {
	return provider == "telegram" || provider == "slack" || provider == "discord"
}

func Validate(provider, secret, chat string) error {
	if len(secret) > 512 {
		return errors.New("Credential is too long")
	}
	if provider == "telegram" {
		if !botToken.MatchString(secret) {
			return errors.New("Enter a valid Telegram bot token from BotFather")
		}
		if !chatID.MatchString(chat) {
			return errors.New("Enter a numeric Telegram chat ID or an @channel username")
		}
		return nil
	}
	u, err := url.Parse(secret)
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || chat != "" {
		return errors.New("Enter the provider's HTTPS incoming webhook URL without query parameters")
	}
	if provider == "slack" && (u.Host == "hooks.slack.com" || u.Host == "hooks.slack-gov.com") && slackPath.MatchString(u.Path) {
		return nil
	}
	if provider == "discord" && u.Host == "discord.com" && discordPath.MatchString(u.Path) {
		return nil
	}
	return errors.New("Enter a Slack or Discord incoming webhook URL for the selected provider")
}

// Resolve and check the address used by the connection, preventing DNS rebinding to private networks.
func publicDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, errors.New("invalid provider address")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, errors.New("provider DNS unavailable")
	}
	for _, addr := range ips {
		ip := addr.IP
		if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		conn, err := (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
	}
	return nil, errors.New("provider connection unavailable")
}

var client = &http.Client{
	Timeout:       3 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	Transport:     &http.Transport{DialContext: publicDial, TLSHandshakeTimeout: 3 * time.Second, MaxIdleConns: 8, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second},
}

func Send(ctx context.Context, provider, secret, chat, text string) error {
	return send(ctx, client, provider, secret, chat, text)
}

func send(ctx context.Context, c *http.Client, provider, secret, chat, text string) error {
	if err := Validate(provider, secret, chat); err != nil {
		return err
	}
	endpoint := secret
	var payload any
	switch provider {
	case "telegram":
		endpoint = "https://api.telegram.org/bot" + secret + "/sendMessage"
		payload = map[string]any{"chat_id": chat, "text": text, "link_preview_options": map[string]bool{"is_disabled": true}}
	case "slack":
		payload = map[string]any{"text": strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text), "mrkdwn": false, "unfurl_links": false, "unfurl_media": false}
	case "discord":
		endpoint += "?wait=true"
		payload = map[string]any{"content": text, "allowed_mentions": map[string]any{"parse": []string{}}}
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return errors.New("Could not prepare provider request")
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.Do(req)
	// Transport errors may contain secret-bearing URLs; never return them.
	if err != nil {
		return errors.New("Could not reach provider; check the connection and try again")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Provider returned HTTP %d; check credentials and destination permissions", res.StatusCode)
	}
	if provider == "telegram" {
		var reply struct {
			OK bool `json:"ok"`
		}
		if json.NewDecoder(io.LimitReader(res.Body, 65536)).Decode(&reply) != nil || !reply.OK {
			return errors.New("Telegram rejected the message; check the bot token and chat permissions")
		}
	}
	return nil
}
