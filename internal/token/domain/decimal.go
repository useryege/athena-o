package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Decimal is Token Intelligence's exact base-10 value type. Its JSON form is
// always a canonical quoted decimal string.
type Decimal struct {
	value decimal.Decimal
}

func ParseDecimal(value string) (Decimal, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Decimal{}, fmt.Errorf("decimal value is empty")
	}
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return Decimal{}, fmt.Errorf("parse decimal %q: %w", value, err)
	}
	if parsed.IsZero() {
		parsed = decimal.Zero
	}
	return Decimal{value: parsed}, nil
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
	if d.value.IsZero() {
		return "0"
	}
	return d.value.String()
}

func (d Decimal) Equal(other Decimal) bool {
	return d.value.Equal(other.value)
}

func (d Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

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
