package tradersync

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Cursor struct {
	Version                                         int
	Kind, PrincipalID, FilterDigest, Direction      string
	PageSize                                        int32
	SnapshotID, UpperID, LowerID, AfterID, Time, ID string
	Empty                                           bool
}

func invalidCursor() error {
	return status.Error(codes.InvalidArgument, "invalid cursor or cursor context")
}
func normalizePageSize(size int32) (int32, error) {
	if size == 0 {
		return 50, nil
	}
	if size < 1 || size > 100 {
		return 0, status.Error(codes.InvalidArgument, "page_size must be between 1 and 100")
	}
	return size, nil
}
func cursorNumber(s string) (int64, error) {
	n, e := strconv.ParseInt(s, 10, 64)
	if e != nil || n < 0 || strconv.FormatInt(n, 10) != s {
		return 0, invalidCursor()
	}
	return n, nil
}
func validateCursor(c Cursor) error {
	if c.Version != 1 || c.PrincipalID == "" || c.FilterDigest == "" || c.PageSize < 1 || c.PageSize > 100 {
		return invalidCursor()
	}
	if c.Kind == "part" {
		if c.Direction != "asc" {
			return invalidCursor()
		}
	} else if c.Direction != "desc" {
		return invalidCursor()
	}
	switch c.Kind {
	case "activity_snapshot", "activity_next", "activity_refresh":
		snapshot, e := cursorNumber(c.SnapshotID)
		if e != nil {
			return e
		}
		if c.Time != "" || c.ID != "" {
			return invalidCursor()
		}
		switch c.Kind {
		case "activity_snapshot":
			if c.UpperID != "" || c.LowerID != "" || c.AfterID != "" || c.Empty {
				return invalidCursor()
			}
		case "activity_next":
			n, e := cursorNumber(c.AfterID)
			if e != nil || n == 0 || n > snapshot || c.Empty || c.UpperID != "" || c.LowerID != "" {
				return invalidCursor()
			}
		case "activity_refresh":
			if c.AfterID != "" {
				return invalidCursor()
			}
			if c.Empty {
				if c.UpperID != "" || c.LowerID != "" {
					return invalidCursor()
				}
				return nil
			}
			upper, e := cursorNumber(c.UpperID)
			if e != nil {
				return e
			}
			lower, e := cursorNumber(c.LowerID)
			if e != nil || lower == 0 || lower > upper || upper > snapshot {
				return invalidCursor()
			}
		}
	case "history", "subscription", "admin_subscription":
		if _, e := time.Parse(time.RFC3339Nano, c.Time); e != nil || c.ID == "" {
			return invalidCursor()
		}
		if c.SnapshotID != "" || c.UpperID != "" || c.LowerID != "" || c.AfterID != "" || c.Empty {
			return invalidCursor()
		}
	case "part":
		n, e := cursorNumber(c.ID)
		if e != nil || n == 0 || c.Time != "" || c.SnapshotID != "" || c.UpperID != "" || c.LowerID != "" || c.AfterID != "" || c.Empty {
			return invalidCursor()
		}
	default:
		return invalidCursor()
	}
	return nil
}
func EncodeCursor(c Cursor, key []byte) (string, error) {
	if len(key) == 0 {
		return "", invalidCursor()
	}
	if e := validateCursor(c); e != nil {
		return "", e
	}
	raw, e := json.Marshal(c)
	if e != nil {
		return "", e
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func DecodeCursor(token string, key []byte, principalID, filterDigest, kind string, pageSize int32) (Cursor, error) {
	var c Cursor
	if len(key) == 0 || len(token) > 8192 {
		return c, invalidCursor()
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return c, invalidCursor()
	}
	raw, e := base64.RawURLEncoding.Strict().DecodeString(parts[0])
	if e != nil {
		return c, invalidCursor()
	}
	sig, e := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if e != nil {
		return c, invalidCursor()
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(raw)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return c, invalidCursor()
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if e = decoder.Decode(&c); e != nil {
		return Cursor{}, invalidCursor()
	}
	if e = decoder.Decode(new(any)); e != io.EOF {
		return Cursor{}, invalidCursor()
	}
	canonical, _ := json.Marshal(c)
	if !bytes.Equal(canonical, raw) {
		return Cursor{}, invalidCursor()
	}
	if e = validateCursor(c); e != nil {
		return Cursor{}, e
	}
	if c.PrincipalID != principalID || c.FilterDigest != filterDigest || c.Kind != kind || c.PageSize != pageSize {
		return Cursor{}, invalidCursor()
	}
	return c, nil
}
