package shared

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type ChainID int64
type ProjectID int64

type Address [20]byte
type Hash [32]byte

type Page struct {
	Number int32
	Size   int32
}

func BytesToAddress(value []byte) Address {
	var result Address
	copyRightAligned(result[:], value)
	return result
}

func BytesToHash(value []byte) Hash {
	var result Hash
	copyRightAligned(result[:], value)
	return result
}

func HexToAddress(value string) (Address, error) {
	var result Address
	if err := decodeFixedHex(value, result[:]); err != nil {
		return Address{}, err
	}
	return result, nil
}

func HexToHash(value string) (Hash, error) {
	var result Hash
	if err := decodeFixedHex(value, result[:]); err != nil {
		return Hash{}, err
	}
	return result, nil
}

func (value Address) Bytes() []byte  { return append([]byte(nil), value[:]...) }
func (value Hash) Bytes() []byte     { return append([]byte(nil), value[:]...) }
func (value Address) Hex() string    { return "0x" + hex.EncodeToString(value[:]) }
func (value Hash) Hex() string       { return "0x" + hex.EncodeToString(value[:]) }
func (value Address) String() string { return value.Hex() }
func (value Hash) String() string    { return value.Hex() }
func (value Address) IsZero() bool   { return value == Address{} }
func (value Hash) IsZero() bool      { return value == Hash{} }

func (value Address) MarshalText() ([]byte, error) { return []byte(value.Hex()), nil }
func (value Hash) MarshalText() ([]byte, error)    { return []byte(value.Hex()), nil }

func (value *Address) UnmarshalText(text []byte) error {
	parsed, err := HexToAddress(string(text))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

func (value *Hash) UnmarshalText(text []byte) error {
	parsed, err := HexToHash(string(text))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

func (value Address) MarshalJSON() ([]byte, error) { return json.Marshal(value.Hex()) }
func (value Hash) MarshalJSON() ([]byte, error)    { return json.Marshal(value.Hex()) }

func (value *Address) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	return value.UnmarshalText([]byte(text))
}

func (value *Hash) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	return value.UnmarshalText([]byte(text))
}

func decodeFixedHex(value string, destination []byte) error {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
	if len(value) != len(destination)*2 {
		return fmt.Errorf("hex value must contain %d bytes", len(destination))
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return fmt.Errorf("decode hex value: %w", err)
	}
	copy(destination, decoded)
	return nil
}

func copyRightAligned(destination, source []byte) {
	if len(source) > len(destination) {
		source = source[len(source)-len(destination):]
	}
	copy(destination[len(destination)-len(source):], source)
}
