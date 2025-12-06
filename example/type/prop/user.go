package prop

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type UserMetadata struct {
	Preferences map[string]any `json:"preferences,omitempty"`
}

func (r *UserMetadata) Scan(value any) error {
	if value == nil {
		return nil
	}
	data, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(data, r)
}

func (r *UserMetadata) Value() (driver.Value, error) {
	return json.Marshal(r)
}
