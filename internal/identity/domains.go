package identity

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidPublicOrigin = errors.New("Enter an HTTPS origin with a DNS hostname and no path, credentials, query or fragment.")

var dnsLabel = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func publicOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" ||
		(u.Path != "" && u.Path != "/") || u.RawPath != "" || strings.ContainsAny(raw, "?#") {
		return "", ErrInvalidPublicOrigin
	}
	host := strings.ToLower(u.Hostname())
	dot := strings.LastIndexByte(host, '.')
	if len(host) > 253 || dot < 0 || dot == len(host)-1 ||
		host[dot+1] < 'a' || host[dot+1] > 'z' || strings.ContainsAny(u.Host, "[]") || strings.HasSuffix(u.Host, ":") {
		return "", ErrInvalidPublicOrigin
	}
	for _, label := range strings.Split(host, ".") {
		if !dnsLabel.MatchString(label) {
			return "", ErrInvalidPublicOrigin
		}
	}
	if port := u.Port(); port != "" {
		number, err := strconv.ParseUint(port, 10, 16)
		if err != nil || number == 0 {
			return "", ErrInvalidPublicOrigin
		}
		if number != 443 {
			host += ":" + strconv.FormatUint(number, 10)
		}
	}
	return "https://" + host, nil
}

func InstanceOrigin(ctx context.Context, db Querier) (string, error) {
	var origin string
	err := db.QueryRow(ctx, "SELECT public_origin FROM instance_domain WHERE singleton").Scan(&origin)
	return origin, err
}

func UpdateInstanceOrigin(ctx context.Context, pool *pgxpool.Pool, actor uuid.UUID, expected, input string) (string, error) {
	origin, err := publicOrigin(input)
	if err != nil {
		return "", err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var operator bool
	err = tx.QueryRow(ctx, "SELECT instance_operator FROM users WHERE id = $1 AND disabled_at IS NULL FOR SHARE", actor).Scan(&operator)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !operator) {
		return "", ErrForbidden
	}
	if err != nil {
		return "", err
	}
	tag, err := tx.Exec(ctx, "UPDATE instance_domain SET public_origin = $1 WHERE singleton AND public_origin = $2", origin, expected)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", ErrConflict
	}
	if origin != expected {
		if err := RecordAuditEvent(ctx, tx, &actor, "instance.domain.updated", map[string]any{"public_origin": origin}); err != nil {
			return "", err
		}
	}
	return origin, tx.Commit(ctx)
}

func SetInstanceOperator(ctx context.Context, pool *pgxpool.Pool, username string, granted bool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	var current bool
	err = tx.QueryRow(ctx, "SELECT id, instance_operator FROM users WHERE username = $1 AND (NOT $2 OR disabled_at IS NULL) FOR UPDATE", username, granted).Scan(&id, &current)
	if err != nil {
		return MapErr(err)
	}
	if current == granted {
		return nil
	}
	if _, err := tx.Exec(ctx, "UPDATE users SET instance_operator = $1 WHERE id = $2", granted, id); err != nil {
		return err
	}
	if err := RecordAuditEvent(ctx, tx, nil, "user.instance_operator.updated", map[string]any{"user_id": id, "granted": granted}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
