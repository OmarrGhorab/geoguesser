package profiles

import "encoding/json"

// OptionalString distinguishes an omitted request field from a field that
// was explicitly provided (including explicit JSON null). It implements
// json.Unmarshaler so the standard decoder only sets Set=true when the key
// is present in the request body.
type OptionalString struct {
	Set   bool
	Value *string
}

// UnmarshalJSON implements json.Unmarshaler.
func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	o.Value = &s
	return nil
}

// OptionalPrefs distinguishes an omitted preferences field from a field
// that was explicitly provided (including explicit JSON null).
type OptionalPrefs struct {
	Set   bool
	Value map[string]any
}

// UnmarshalJSON implements json.Unmarshaler.
func (o *OptionalPrefs) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	o.Value = m
	return nil
}
