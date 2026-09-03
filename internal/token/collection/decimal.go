package collection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Decimal is Token Intelligence's exact base-10 value type. Its JSON form is
// always a canonical quoted decimal string.
type Decimal struct {
	canonical string
}

var decimalPattern = regexp.MustCompile(`^([+-]?)([0-9]*)(?:\.([0-9]*))?(?:[eE]([+-]?[0-9]+))?$`)

func ParseDecimal(value string) (Decimal, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Decimal{}, fmt.Errorf("decimal value is empty")
	}
	canonical, err := canonicalDecimal(value)
	if err != nil {
		return Decimal{}, fmt.Errorf("parse decimal %q: %w", value, err)
	}
	return Decimal{canonical: canonical}, nil
}

func ParseOptionalDecimal(value string) (*Decimal, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := ParseDecimal(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (d Decimal) String() string {
	if d.canonical == "" {
		return "0"
	}
	return d.canonical
}

func (d Decimal) Equal(other Decimal) bool { return d.String() == other.String() }

func (d Decimal) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

func (d *Decimal) UnmarshalJSON(data []byte) error {
	if d == nil {
		return fmt.Errorf("unmarshal decimal into nil receiver")
	}
	if bytes.Equal(data, []byte("null")) {
		return fmt.Errorf("decimal null requires a pointer field")
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decimal must be a quoted string: %w", err)
	}
	parsed, err := ParseDecimal(value)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func canonicalDecimal(value string) (string, error) {
	parts := decimalPattern.FindStringSubmatch(value)
	if parts == nil || parts[2] == "" && parts[3] == "" {
		return "", fmt.Errorf("invalid decimal syntax")
	}
	exponent := int64(0)
	if parts[4] != "" {
		parsed, err := strconv.ParseInt(parts[4], 10, 32)
		if err != nil {
			return "", fmt.Errorf("invalid decimal exponent: %w", err)
		}
		exponent = parsed
	}
	digits := parts[2] + parts[3]
	decimalPosition := int64(len(parts[2])) + exponent
	leadingZeros := len(digits) - len(strings.TrimLeft(digits, "0"))
	if leadingZeros == len(digits) {
		return "0", nil
	}
	digits = digits[leadingZeros:]
	decimalPosition -= int64(leadingZeros)
	digits = strings.TrimRight(digits, "0")

	var canonical string
	switch {
	case decimalPosition <= 0:
		canonical = "0." + strings.Repeat("0", int(-decimalPosition)) + digits
	case decimalPosition >= int64(len(digits)):
		canonical = digits + strings.Repeat("0", int(decimalPosition)-len(digits))
	default:
		canonical = digits[:decimalPosition] + "." + digits[decimalPosition:]
	}
	if parts[1] == "-" {
		canonical = "-" + canonical
	}
	return canonical, nil
}
