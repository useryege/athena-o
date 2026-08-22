package appbootstrap

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON keeps the HTTP representation aligned with the protobuf and
// Swagger enum names instead of encoding the generated int32 value.
func (s AppBootstrapSessionStatus) MarshalJSON() ([]byte, error) {
	name, ok := AppBootstrapSessionStatus_name[int32(s)]
	if !ok {
		return nil, fmt.Errorf("unknown app bootstrap session status %d", s)
	}
	return json.Marshal(name)
}

// UnmarshalJSON decodes the named HTTP representation of a bootstrap status.
func (s *AppBootstrapSessionStatus) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	value, ok := AppBootstrapSessionStatus_value[name]
	if !ok {
		return fmt.Errorf("unknown app bootstrap session status %q", name)
	}
	*s = AppBootstrapSessionStatus(value)
	return nil
}
