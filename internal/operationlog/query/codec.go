package query

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Viewer struct {
	AccountID      string
	Realm          string
	CredentialKind string
	SessionBinding []byte
	AccessRevision uint64
}
type CursorPosition struct {
	StartedAt   time.Time `json:"startedAt"`
	OperationID string    `json:"operationId"`
}
type CursorData struct {
	Filters          NormalizedFilter
	Position         CursorPosition
	IssuedAt         time.Time
	ExpiresAt        time.Time
	SnapshotSequence int64
	SnapshotAt       time.Time
}
type SnapshotData struct {
	Filters          NormalizedFilter
	IssuedAt         time.Time
	ExpiresAt        time.Time
	SnapshotSequence int64
	SnapshotAt       time.Time
}
type Codec struct {
	key []byte
	now func() time.Time
}
type wireToken struct {
	Version  int               `json:"v"`
	Kind     string            `json:"k"`
	Filters  *NormalizedFilter `json:"f,omitempty"`
	Position *CursorPosition   `json:"p,omitempty"`
	W        int64             `json:"w"`
	At       time.Time         `json:"at"`
	Issued   time.Time         `json:"i"`
	Expires  time.Time         `json:"e"`
	Viewer   string            `json:"b"`
	Sig      string            `json:"s"`
}

func NewCodec(key []byte, now func() time.Time) (*Codec, error) {
	if len(key) != 32 {
		return nil, errors.New("cursor HMAC key must be 32 bytes")
	}
	if now == nil {
		now = time.Now
	}
	return &Codec{key: append([]byte(nil), key...), now: now}, nil
}
func (c *Codec) EncodeCursor(f NormalizedFilter, viewer Viewer, pos CursorPosition, now time.Time) (string, error) {
	return c.EncodeCursorAt(f, viewer, pos, 0, time.Time{}, now)
}
func (c *Codec) EncodeCursorAt(f NormalizedFilter, viewer Viewer, pos CursorPosition, sequence int64, snapshotAt, now time.Time) (string, error) {
	return c.encode("cursor", &f, &pos, viewer, sequence, snapshotAt, now)
}
func (c *Codec) EncodeSnapshot(f NormalizedFilter, viewer Viewer, now time.Time) (string, error) {
	return c.encode("snapshot", &f, nil, viewer, 0, time.Time{}, now)
}
func (c *Codec) EncodeSnapshotAt(f NormalizedFilter, viewer Viewer, sequence int64, at, now time.Time) (string, error) {
	return c.encode("snapshot", &f, nil, viewer, sequence, at, now)
}
func (c *Codec) encode(kind string, f *NormalizedFilter, p *CursorPosition, v Viewer, w int64, at, now time.Time) (string, error) {
	if err := validateViewer(v); err != nil {
		return "", err
	}
	now = now.UTC()
	if at.IsZero() {
		at = now
	}
	x := wireToken{Version: 1, Kind: kind, Filters: f, Position: p, W: w, At: at.UTC(), Issued: now, Expires: now.Add(CursorTTL), Viewer: viewerBinding(v)}
	raw, _ := json.Marshal(x)
	x.Sig = sign(c.key, raw)
	return base64.RawURLEncoding.EncodeToString(mustJSON(x)), nil
}
func (c *Codec) DecodeCursor(token string, viewer Viewer, now time.Time) (CursorData, error) {
	x, err := c.decode(token, "cursor", viewer, now)
	if err != nil {
		return CursorData{}, err
	}
	if x.Filters == nil || x.Position == nil {
		return CursorData{}, ErrCursorInvalid
	}
	return CursorData{Filters: *x.Filters, Position: *x.Position, IssuedAt: x.Issued, ExpiresAt: x.Expires, SnapshotSequence: x.W, SnapshotAt: x.At}, nil
}
func (c *Codec) DecodeSnapshot(token string, viewer Viewer, now time.Time) (SnapshotData, error) {
	x, err := c.decode(token, "snapshot", viewer, now)
	if err != nil {
		return SnapshotData{}, err
	}
	if x.Filters == nil {
		return SnapshotData{}, ErrCursorInvalid
	}
	return SnapshotData{Filters: *x.Filters, IssuedAt: x.Issued, ExpiresAt: x.Expires, SnapshotSequence: x.W, SnapshotAt: x.At}, nil
}
func (c *Codec) decode(token, kind string, v Viewer, now time.Time) (wireToken, error) {
	if len(token) == 0 || len(token) > 4096 {
		return wireToken{}, ErrCursorInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return wireToken{}, ErrCursorInvalid
	}
	var x wireToken
	if json.Unmarshal(raw, &x) != nil || x.Version != 1 || x.Kind != kind || x.Sig == "" {
		return wireToken{}, ErrCursorInvalid
	}
	sig := x.Sig
	x.Sig = ""
	unsigned, _ := json.Marshal(x)
	if !hmac.Equal([]byte(sig), []byte(sign(c.key, unsigned))) {
		return wireToken{}, ErrCursorInvalid
	}
	if x.Viewer != viewerBinding(v) {
		return wireToken{}, ErrCursorInvalid
	}
	now = now.UTC()
	if !now.Before(x.Expires) {
		return wireToken{}, ErrCursorExpired
	}
	if now.Before(x.Issued) {
		return wireToken{}, ErrCursorInvalid
	}
	return x, nil
}
func (c *Codec) ValidateCursorParameters(token string, f NormalizedFilter, viewer Viewer, now time.Time) error {
	x, err := c.decode(token, "cursor", viewer, now)
	if err != nil {
		return err
	}
	if x.Filters == nil || !filtersEqual(*x.Filters, f) {
		return ErrCursorInvalid
	}
	return nil
}
func filtersEqual(a, b NormalizedFilter) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return hmac.Equal(aj, bj)
}
func validateViewer(v Viewer) error {
	if strings.TrimSpace(v.AccountID) == "" || v.Realm != "ADMIN" || (v.CredentialKind != "LOGIN_SESSION" && v.CredentialKind != "DEVELOPMENT") || len(v.SessionBinding) != 32 || v.AccessRevision == 0 {
		return ErrCursorInvalid
	}
	return nil
}
func viewerBinding(v Viewer) string {
	h := sha256.New()
	h.Write([]byte(v.AccountID))
	h.Write([]byte{0})
	h.Write([]byte(v.Realm))
	h.Write([]byte{0})
	h.Write([]byte(v.CredentialKind))
	h.Write([]byte{0})
	h.Write(v.SessionBinding)
	h.Write([]byte(fmt.Sprintf("%d", v.AccessRevision)))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func sign(key, raw []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(raw)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
