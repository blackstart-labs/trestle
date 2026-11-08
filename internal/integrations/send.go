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
	"strconv"
	"strings"
	"time"
)

var (
	botToken    = regexp.MustCompile(`^[0-9]{1,20}:[A-Za-z0-9_-]{20,100}$`)
	chatID      = regexp.MustCompile(`^(-?[0-9]{1,20}|@[A-Za-z0-9_]{5,32})$`)
	topicIDRe   = regexp.MustCompile(`^[0-9]{1,20}$`)
	slackPath   = regexp.MustCompile(`^/services/[A-Za-z0-9]+/[A-Za-z0-9]+/[A-Za-z0-9]+$`)
	discordPath = regexp.MustCompile(`^/api(?:/v[0-9]+)?/webhooks/[0-9]+/[A-Za-z0-9_-]+$`)
)

func ValidProvider(provider string) bool {
	return provider == "telegram" || provider == "slack" || provider == "discord"
}

func Validate(provider, secret, chat string) error {
	return ValidateWithTopic(provider, secret, chat, "")
}

func ValidateWithTopic(provider, secret, chat, topic string) error {
	if len(secret) > 512 {
		return errors.New("Credential is too long")
	}
	if provider == "telegram" {
		if !botToken.MatchString(secret) {
			return errors.New("Enter a valid Telegram bot token from BotFather")
		}
		cleanChat := strings.TrimSpace(chat)
		cleanTopic := strings.TrimSpace(topic)
		if cleanTopic == "" {
			if idx := strings.IndexAny(cleanChat, "/:"); idx != -1 && !strings.HasPrefix(cleanChat, "@") {
				cleanTopic = cleanChat[idx+1:]
				cleanChat = cleanChat[:idx]
			}
		}
		if !chatID.MatchString(cleanChat) {
			return errors.New("Enter a numeric Telegram chat ID or an @channel username")
		}
		if cleanTopic != "" && !topicIDRe.MatchString(cleanTopic) {
			return errors.New("Enter a numeric Telegram topic ID")
		}
		return nil
	}
	u, err := url.Parse(secret)
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || chat != "" || topic != "" {
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
	return SendWithTopic(ctx, provider, secret, chat, "", text)
}

func SendWithTopic(ctx context.Context, provider, secret, chat, topic, text string) error {
	return send(ctx, client, provider, secret, chat, topic, text)
}

func send(ctx context.Context, c *http.Client, provider, secret, chat, topic, text string) error {
	if err := ValidateWithTopic(provider, secret, chat, topic); err != nil {
		return err
	}
	cleanChat := strings.TrimSpace(chat)
	cleanTopic := strings.TrimSpace(topic)
	if cleanTopic == "" {
		if idx := strings.IndexAny(cleanChat, "/:"); idx != -1 && !strings.HasPrefix(cleanChat, "@") {
			cleanTopic = cleanChat[idx+1:]
			cleanChat = cleanChat[:idx]
		}
	}
	endpoint := secret
	var payload any
	switch provider {
	case "telegram":
		endpoint = "https://api.telegram.org/bot" + secret + "/sendMessage"
		p := map[string]any{
			"chat_id": cleanChat,
			"text": text,
			"link_preview_options": map[string]bool{"is_disabled": true},
		}
		if cleanTopic != "" {
			if tid, err := strconv.ParseInt(cleanTopic, 10, 64); err == nil && tid > 0 {
				p["message_thread_id"] = tid
			}
		}
		if strings.Contains(text, "<a href=") {
			p["parse_mode"] = "HTML"
		}
		payload = p
	case "slack":
		slackText := text
		mrkdwn := false
		if strings.Contains(text, "|Open in Trestle>") {
			mrkdwn = true
		} else {
			slackText = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(text)
		}
		payload = map[string]any{"text": slackText, "mrkdwn": mrkdwn, "unfurl_links": false, "unfurl_media": false}
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
	if provider == "telegram" && (res.StatusCode < 200 || res.StatusCode >= 300) && strings.Contains(text, "<a href=") {
		// Fallback without parse_mode if Telegram rejects HTML entities
		plainText := strings.ReplaceAll(text, "<a href=\"", "")
		plainText = strings.ReplaceAll(plainText, "\">Open in Trestle</a>", "")
		p := map[string]any{
			"chat_id": cleanChat,
			"text": plainText,
			"link_preview_options": map[string]bool{"is_disabled": true},
		}
		if cleanTopic != "" {
			if tid, err := strconv.ParseInt(cleanTopic, 10, 64); err == nil && tid > 0 {
				p["message_thread_id"] = tid
			}
		}
		fallbackBody, _ := json.Marshal(p)
		if fbReq, fbErr := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(fallbackBody)); fbErr == nil {
			fbReq.Header.Set("Content-Type", "application/json")
			if fbRes, fbDoErr := c.Do(fbReq); fbDoErr == nil {
				defer fbRes.Body.Close()
				if fbRes.StatusCode >= 200 && fbRes.StatusCode < 300 {
					var reply struct {
						OK bool `json:"ok"`
					}
					if json.NewDecoder(io.LimitReader(fbRes.Body, 65536)).Decode(&reply) == nil && reply.OK {
						return nil
					}
				}
			}
		}
	}
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
