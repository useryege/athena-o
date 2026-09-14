package account

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// MarshalJSON keeps the HTTP representation aligned with the protobuf and
// Swagger enum names instead of encoding the generated int32 value.
func (a AccountDataAccess) MarshalJSON() ([]byte, error) {
	name, ok := AccountDataAccess_name[int32(a)]
	if !ok {
		return nil, fmt.Errorf("unknown account data access %d", a)
	}
	return json.Marshal(name)
}

// UnmarshalJSON accepts both the protobuf enum name and its numeric value.
func (a *AccountDataAccess) UnmarshalJSON(data []byte) error {
	value, err := unmarshalAccountEnum(data, AccountDataAccess_value, "account data access")
	if err != nil {
		return err
	}
	*a = AccountDataAccess(value)
	return nil
}

// MarshalJSON keeps the HTTP representation aligned with the protobuf and
// Swagger enum names instead of encoding the generated int32 value.
func (m AccountDataModule) MarshalJSON() ([]byte, error) {
	name, ok := AccountDataModule_name[int32(m)]
	if !ok {
		return nil, fmt.Errorf("unknown account data module %d", m)
	}
	return json.Marshal(name)
}

// UnmarshalJSON accepts both the protobuf enum name and its numeric value.
func (m *AccountDataModule) UnmarshalJSON(data []byte) error {
	value, err := unmarshalAccountEnum(data, AccountDataModule_value, "account data module")
	if err != nil {
		return err
	}
	*m = AccountDataModule(value)
	return nil
}

func unmarshalAccountEnum(data []byte, namedValues map[string]int32, description string) (int32, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return 0, fmt.Errorf("empty %s", description)
	}
	if data[0] == '"' {
		var name string
		if err := json.Unmarshal(data, &name); err != nil {
			return 0, err
		}
		value, ok := namedValues[name]
		if !ok {
			return 0, fmt.Errorf("unknown %s %q", description, name)
		}
		return value, nil
	}

	value, err := strconv.ParseInt(string(data), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", description, data, err)
	}
	return int32(value), nil
}

// MarshalJSON follows protobuf JSON conventions for uint64 while preserving
// the existing camelCase account-access field names.
func (a AccountAccess) MarshalJSON() ([]byte, error) {
	revision := ""
	if a.Revision != 0 {
		revision = strconv.FormatUint(a.Revision, 10)
	}
	return json.Marshal(struct {
		LoginEnabled         bool                   `json:"loginEnabled,omitempty"`
		ApiKeyEnabled        bool                   `json:"apiKeyEnabled,omitempty"`
		ProfitSharingEnabled bool                   `json:"profitSharingEnabled,omitempty"`
		Revision             string                 `json:"revision,omitempty"`
		ModuleAccess         []*AccountModuleAccess `json:"moduleAccess,omitempty"`
	}{
		LoginEnabled:         a.LoginEnabled,
		ApiKeyEnabled:        a.ApiKeyEnabled,
		ProfitSharingEnabled: a.ProfitSharingEnabled,
		Revision:             revision,
		ModuleAccess:         a.ModuleAccess,
	})
}

// UnmarshalJSON accepts revision as either its protobuf JSON string form or a
// numeric value used by older handwritten HTTP clients.
func (a *AccountAccess) UnmarshalJSON(data []byte) error {
	var value struct {
		LoginEnabled         bool                   `json:"loginEnabled,omitempty"`
		ApiKeyEnabled        bool                   `json:"apiKeyEnabled,omitempty"`
		ProfitSharingEnabled bool                   `json:"profitSharingEnabled,omitempty"`
		Revision             json.RawMessage        `json:"revision,omitempty"`
		ModuleAccess         []*AccountModuleAccess `json:"moduleAccess,omitempty"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	revision, err := unmarshalUint64StringOrNumber(value.Revision)
	if err != nil {
		return fmt.Errorf("invalid account access revision: %w", err)
	}
	*a = AccountAccess{
		LoginEnabled:         value.LoginEnabled,
		ApiKeyEnabled:        value.ApiKeyEnabled,
		ProfitSharingEnabled: value.ProfitSharingEnabled,
		Revision:             revision,
		ModuleAccess:         value.ModuleAccess,
	}
	return nil
}

func unmarshalUint64StringOrNumber(data []byte) (uint64, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return 0, nil
	}
	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return 0, err
		}
		return strconv.ParseUint(value, 10, 64)
	}
	return strconv.ParseUint(string(data), 10, 64)
}
